import json
import os
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse


def _read_json(handler: BaseHTTPRequestHandler):
    length = int(handler.headers.get("Content-Length", "0") or "0")
    if length <= 0:
        return None
    raw = handler.rfile.read(length)
    return json.loads(raw.decode("utf-8"))


def _write_json(handler: BaseHTTPRequestHandler, status: int, payload):
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json; charset=utf-8")
    handler.send_header("Content-Length", str(len(body)))
    handler.end_headers()
    handler.wfile.write(body)


def _team_codes(n: int):
    # Volleyball defaults:
    # 10..14 => 2 teams (5..7)
    # 15..18 => 3 teams (5..6)
    if n > 14:
        return ["A", "B", "C"]
    return ["A", "B"]


def _capacities(n: int, codes):
    base = n // len(codes)
    rest = n % len(codes)
    out = {}
    for idx, code in enumerate(codes):
        out[code] = base + (1 if idx < rest else 0)
    return out


def _normalize_role(role: str):
    role = (role or "").strip().lower()
    if role in ("setter", "libero", "attacker"):
        return role
    return ""


def split_teams(req: dict):
    players = req.get("players") or []
    relations_in = req.get("relations") or []
    team_codes = req.get("teamCodes")
    capacity = req.get("capacity")

    # Normalize players
    norm_players = []
    for p in players:
        user_id = int(p.get("userID") or 0)
        if user_id == 0:
            continue
        rating = float(p.get("rating") or 5.0)
        role = _normalize_role(p.get("playerType") or "")
        norm_players.append(
            {
                "userID": user_id,
                "rating": rating,
                "playerType": role,
            }
        )

    n = len(norm_players)
    if n == 0:
        return {"assignments": []}

    if team_codes is None:
        team_codes = _team_codes(n)
    if capacity is None:
        capacity = _capacities(n, team_codes)

    # Build relation graph (undirected)
    rel = {}
    for e in relations_in:
        a = int(e.get("userAID") or 0)
        b = int(e.get("userBID") or 0)
        if a == 0 or b == 0:
            continue
        t = (e.get("relationType") or "").strip().lower()
        if t not in ("prefer_together", "avoid_together"):
            continue
        w = int(e.get("weight") or 0)
        if w <= 0:
            continue
        rel.setdefault(a, []).append({"other": b, "type": t, "weight": w})
        rel.setdefault(b, []).append({"other": a, "type": t, "weight": w})

    buckets = []
    by_code = {}
    for code in team_codes:
        b = {"code": code, "players": [], "score": 0.0, "setters": 0, "liberos": 0}
        buckets.append(b)
        by_code[code] = b

    assigned_team = {}

    def assign_to_best(player: dict, prefer_role: bool):
        chosen = None
        best = None
        role = player.get("playerType") or ""
        for b in buckets:
            code = b["code"]
            if len(b["players"]) >= int(capacity.get(code, 0)):
                continue
            role_penalty = 0.0
            if prefer_role:
                if role == "setter":
                    role_penalty = float(b["setters"]) * 3.0
                elif role == "libero":
                    role_penalty = float(b["liberos"]) * 3.0
            relation_penalty = 0.0
            for edge in rel.get(player["userID"], []):
                other_team = assigned_team.get(edge["other"])
                if other_team is None:
                    continue
                if edge["type"] == "prefer_together":
                    if other_team != code:
                        relation_penalty += float(edge["weight"]) * 4.0
                    else:
                        relation_penalty -= float(edge["weight"]) * 0.75
                elif edge["type"] == "avoid_together":
                    if other_team == code:
                        relation_penalty += float(edge["weight"]) * 6.0
            metric = b["score"] + role_penalty + relation_penalty + float(len(b["players"])) * 0.25
            if chosen is None or metric < best:
                chosen = b
                best = metric
        if chosen is None:
            chosen = buckets[0]
        chosen["players"].append(player)
        chosen["score"] += float(player["rating"])
        assigned_team[player["userID"]] = chosen["code"]
        if role == "setter":
            chosen["setters"] += 1
        if role == "libero":
            chosen["liberos"] += 1

    setters = [p for p in norm_players if p["playerType"] == "setter"]
    liberos = [p for p in norm_players if p["playerType"] == "libero"]
    rest = [p for p in norm_players if p["playerType"] not in ("setter", "libero")]

    setters.sort(key=lambda x: x["rating"], reverse=True)
    liberos.sort(key=lambda x: x["rating"], reverse=True)
    rest.sort(key=lambda x: x["rating"], reverse=True)

    for p in setters:
        assign_to_best(p, True)
    for p in liberos:
        assign_to_best(p, True)
    for p in rest:
        assign_to_best(p, False)

    assignments = []
    for code in team_codes:
        team_players = list(by_code[code]["players"])
        team_players.sort(key=lambda x: x["rating"], reverse=True)
        for pos, p in enumerate(team_players):
            assignments.append({"userID": p["userID"], "team": code, "position": pos})

    return {"assignments": assignments, "teamCodes": team_codes, "capacity": capacity}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        # Keep stdout clean in docker logs
        return

    def do_GET(self):
        path = urlparse(self.path).path
        if path == "/healthz":
            _write_json(self, 200, {"status": "ok", "time": int(time.time())})
            return
        _write_json(self, 404, {"error": "not found"})

    def do_POST(self):
        path = urlparse(self.path).path
        if path != "/split":
            _write_json(self, 404, {"error": "not found"})
            return
        try:
            payload = _read_json(self) or {}
        except Exception as e:
            _write_json(self, 400, {"error": f"invalid json: {e}"})
            return
        try:
            out = split_teams(payload)
        except Exception as e:
            _write_json(self, 500, {"error": str(e)})
            return
        _write_json(self, 200, out)


def main():
    addr = os.environ.get("ML_ADDR", ":9000").strip()
    if addr.startswith(":"):
        host = "0.0.0.0"
        port = int(addr[1:])
    else:
        host, port_s = addr.rsplit(":", 1)
        port = int(port_s)
    server = HTTPServer((host, port), Handler)
    print(f"team split service started on {host}:{port}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()

