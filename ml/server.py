import json
import math
import os
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse


SKILL_CODES = ("receive", "serve", "set", "defense", "attack", "block")
ROLE_TARGET_ORDER = ("setter", "libero", "central")
ROLE_PRIORITY = {
    "setter": 0,
    "central": 1,
    "libero": 2,
    "attacker": 3,
}
SKILL_WEIGHT = {
    "attack": 0.45,
    "block": 0.48,
    "set": 0.42,
    "receive": 0.37,
    "defense": 0.33,
    "serve": 0.28,
}
SLOT_ORDER = (2, 3, 4, 5, 1, 6)


def _to_int(value, default=0):
    try:
        return int(value)
    except Exception:
        return default


def _to_float(value, default=0.0):
    try:
        return float(value)
    except Exception:
        return default


def _normalize_role(role: str):
    role = (role or "").strip().lower()
    if role in ("setter", "libero", "attacker", "central"):
        return role
    if role in ("middle", "middle_blocker", "blocker"):
        return "central"
    if role in ("outside", "opposite", "hitter"):
        return "attacker"
    return ""


def _normalize_relation_type(value: str):
    rel = (value or "").strip().lower()
    if rel in ("prefer_together", "avoid_together"):
        return rel
    return ""


def _normalize_skills(raw):
    out = {code: 5.0 for code in SKILL_CODES}
    if not isinstance(raw, dict):
        return out
    for key, value in raw.items():
        code = (str(key) if key is not None else "").strip().lower()
        if code not in out:
            continue
        parsed = _to_float(value, 0.0)
        if parsed > 0:
            out[code] = parsed
    return out


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


def _sanitize_team_codes(raw_codes, n: int):
    if not isinstance(raw_codes, list):
        return _team_codes(n)
    seen = set()
    out = []
    for raw in raw_codes:
        code = (str(raw) if raw is not None else "").strip().upper()
        if not code or code in seen:
            continue
        out.append(code)
        seen.add(code)
    if len(out) >= 2:
        return out
    return _team_codes(n)


def _capacities(n: int, codes):
    if not codes:
        return {}
    base = n // len(codes)
    rest = n % len(codes)
    out = {}
    for idx, code in enumerate(codes):
        out[code] = base + (1 if idx < rest else 0)
    return out


def _sanitize_capacity(raw_capacity, n: int, team_codes):
    if not isinstance(raw_capacity, dict):
        return _capacities(n, team_codes)
    out = {}
    total = 0
    for code in team_codes:
        value = raw_capacity.get(code)
        if value is None:
            value = raw_capacity.get(code.lower())
        parsed = _to_int(value, 0)
        if parsed < 0:
            parsed = 0
        out[code] = parsed
        total += parsed
    if total < n and team_codes:
        idx = 0
        while total < n:
            code = team_codes[idx % len(team_codes)]
            out[code] += 1
            total += 1
            idx += 1
    elif total > n and team_codes:
        idx = 0
        safe_guard = 0
        # Keep total aligned with roster size, avoid negative capacities.
        while total > n and safe_guard < 10000:
            code = team_codes[idx % len(team_codes)]
            if out[code] > 0:
                out[code] -= 1
                total -= 1
            idx += 1
            safe_guard += 1
    return out


def _sanitize_role_targets(raw_targets, team_codes):
    out = {
        role: {code: 0 for code in team_codes}
        for role in ROLE_TARGET_ORDER
    }
    if not isinstance(raw_targets, dict):
        return out
    for raw_role, by_team in raw_targets.items():
        role = _normalize_role(raw_role)
        if role not in out:
            continue
        if not isinstance(by_team, dict):
            continue
        for code in team_codes:
            value = by_team.get(code)
            if value is None:
                value = by_team.get(code.lower())
            parsed = _to_int(value, 0)
            if parsed < 0:
                parsed = 0
            out[role][code] = parsed
    return out


def _distribute_role_targets(total: int, max_per_team: int, team_codes):
    out = {code: 0 for code in team_codes}
    if total <= 0 or max_per_team <= 0 or not team_codes:
        return out
    remaining = total
    while remaining > 0:
        progressed = False
        for code in team_codes:
            if remaining == 0:
                break
            if out[code] >= max_per_team:
                continue
            out[code] += 1
            remaining -= 1
            progressed = True
        if not progressed:
            break
    return out


def _build_role_targets(units, team_codes, raw_targets):
    if isinstance(raw_targets, dict):
        return _sanitize_role_targets(raw_targets, team_codes)
    count_by_role = {role: 0 for role in ROLE_TARGET_ORDER}
    for unit in units:
        role = unit.get("role", "")
        if role in count_by_role:
            count_by_role[role] += 1
    return {
        "setter": _distribute_role_targets(
            min(count_by_role["setter"], len(team_codes)),
            1,
            team_codes,
        ),
        "libero": _distribute_role_targets(
            min(count_by_role["libero"], len(team_codes)),
            1,
            team_codes,
        ),
        "central": _distribute_role_targets(
            min(count_by_role["central"], 2 * len(team_codes)),
            2,
            team_codes,
        ),
    }


def _player_role(player, player_by_id):
    role = _normalize_role(player.get("playerType", ""))
    if role:
        return role
    user_id = _to_int(player.get("userID"), 0)
    if user_id > 0:
        known = player_by_id.get(user_id)
        if known:
            fallback = _normalize_role(known.get("playerType", ""))
            if fallback:
                return fallback
    owner_id = _to_int(player.get("guestOwnerID"), 0)
    if owner_id > 0:
        owner = player_by_id.get(owner_id)
        if owner:
            inherited = _normalize_role(owner.get("playerType", ""))
            if inherited:
                return inherited
    return ""


def _player_skills(player):
    skills = player.get("skills")
    return _normalize_skills(skills)


def _player_power(role: str, skills):
    receive = skills["receive"]
    serve = skills["serve"]
    set_score = skills["set"]
    defense = skills["defense"]
    attack = skills["attack"]
    block = skills["block"]

    power = (
        attack * 1.15
        + block * 1.12
        + set_score * 1.02
        + receive * 0.95
        + defense * 0.88
        + serve * 0.78
    )
    if role == "setter":
        power += set_score * 0.48 + serve * 0.12
    elif role == "libero":
        power += receive * 0.42 + defense * 0.38
    elif role == "central":
        power += block * 0.56 + attack * 0.18
    elif role == "attacker":
        power += attack * 0.34 + serve * 0.11
    return power


def _new_unit(unit_id: int, players, player_by_id, relations):
    role = _player_role(players[0], player_by_id) if players else ""
    skill_totals = {code: 0.0 for code in SKILL_CODES}
    power_total = 0.0
    relation_weight = 0
    for player in players:
        p_role = _player_role(player, player_by_id)
        p_skills = _player_skills(player)
        for code in SKILL_CODES:
            skill_totals[code] += p_skills[code]
        power_total += _player_power(p_role, p_skills)
        for edge in relations.get(player["userID"], []):
            relation_weight += edge["weight"]
    return {
        "id": unit_id,
        "role": role,
        "players": list(players),
        "size": len(players),
        "skillTotals": skill_totals,
        "powerTotal": power_total,
        "relationWeight": relation_weight,
    }


def _build_units(players, player_by_id, relations):
    owners = []
    guests_by_owner = {}
    standalone_guests = []
    for player in players:
        owner_id = _to_int(player.get("guestOwnerID"), 0)
        if owner_id > 0:
            guests_by_owner.setdefault(owner_id, []).append(player)
            continue
        if player["userID"] < 0:
            standalone_guests.append(player)
            continue
        owners.append(player)

    owners.sort(key=lambda p: p["userID"])
    standalone_guests.sort(key=lambda p: p["userID"])

    units = []
    for owner in owners:
        owner_id = owner["userID"]
        unit_players = [owner]
        guests = guests_by_owner.get(owner_id, [])
        guests.sort(key=lambda p: p["userID"])
        unit_players.extend(guests)
        units.append(_new_unit(owner_id, unit_players, player_by_id, relations))
        if owner_id in guests_by_owner:
            del guests_by_owner[owner_id]

    # Defensive fallback for inconsistent data (guest without owner in roster).
    for owner_id in sorted(guests_by_owner.keys()):
        guests = guests_by_owner.get(owner_id, [])
        guests.sort(key=lambda p: p["userID"])
        for guest in guests:
            unit_id = owner_id if owner_id > 0 else guest["userID"]
            units.append(_new_unit(unit_id, [guest], player_by_id, relations))

    for guest in standalone_guests:
        units.append(_new_unit(guest["userID"], [guest], player_by_id, relations))
    return units


def _build_stat_targets(units, team_codes, capacity):
    total_players = 0
    total_power = 0.0
    total_skills = {code: 0.0 for code in SKILL_CODES}
    for unit in units:
        total_players += unit["size"]
        total_power += unit["powerTotal"]
        for code in SKILL_CODES:
            total_skills[code] += unit["skillTotals"][code]

    if total_players <= 0:
        return {}, {}

    skill_targets = {}
    power_targets = {}
    for code in team_codes:
        ratio = float(capacity.get(code, 0)) / float(total_players)
        skill_targets[code] = {
            skill: total_skills[skill] * ratio
            for skill in SKILL_CODES
        }
        power_targets[code] = total_power * ratio
    return skill_targets, power_targets


def _role_priority(role: str):
    return ROLE_PRIORITY.get(role, 4)


def _placement_score(
    unit,
    bucket,
    role_targets,
    skill_target,
    power_target,
    relations,
    assigned_team_by_user,
):
    penalty = 0.0
    next_players = bucket["playerCount"] + unit["size"]
    overflow = next_players - bucket["capacity"]
    if overflow > 0:
        penalty += float(overflow) * 60.0
    penalty += abs(float(next_players - bucket["capacity"])) * 0.9

    for role in ROLE_TARGET_ORDER:
        target = role_targets.get(role, {}).get(bucket["code"], 0)
        current = bucket["roleCounts"].get(role, 0)
        add = 1 if unit["role"] == role else 0
        nxt = current + add
        if nxt < target:
            penalty += float(target - nxt) * 52.0
        if nxt > target:
            excess_weight = 4.5 if role == "central" else 6.0
            penalty += float(nxt - target) * excess_weight

    next_power = bucket["powerTotal"] + unit["powerTotal"]
    penalty += abs(next_power - power_target) * 0.42

    # Keep strongest centrals opposite strongest attackers.
    if unit.get("role") == "central":
        central_strength = _to_float(unit.get("skillTotals", {}).get("block"), 0.0) * 1.1 + _to_float(
            unit.get("skillTotals", {}).get("attack"), 0.0
        ) * 0.35
        penalty += central_strength * _bucket_attacker_load(bucket) * 0.08
    elif unit.get("role") == "attacker":
        attacker_strength = _to_float(unit.get("skillTotals", {}).get("attack"), 0.0)
        penalty += attacker_strength * _bucket_central_load(bucket) * 0.08

    for code, target in skill_target.items():
        next_skill = bucket["skillTotals"].get(code, 0.0) + unit["skillTotals"].get(code, 0.0)
        penalty += abs(next_skill - target) * SKILL_WEIGHT.get(code, 0.3)

    for player in unit["players"]:
        for edge in relations.get(player["userID"], []):
            other_team = assigned_team_by_user.get(edge["other"])
            if other_team is None:
                continue
            weight = float(edge["weight"])
            if edge["type"] == "prefer_together":
                if other_team != bucket["code"]:
                    penalty += weight * 9.0
                else:
                    penalty -= weight * 1.4
            elif edge["type"] == "avoid_together":
                if other_team == bucket["code"]:
                    penalty += weight * 11.0
                else:
                    penalty -= weight * 0.45
    return penalty


def _choose_bucket_for_unit(
    unit,
    team_codes,
    buckets,
    role_targets,
    skill_targets,
    power_targets,
    relations,
    assigned_team_by_user,
):
    chosen = None
    best = math.inf
    candidates = [
        buckets[code]
        for code in team_codes
        if buckets[code]["playerCount"] + unit["size"] <= buckets[code]["capacity"]
    ]
    if not candidates:
        candidates = [buckets[code] for code in team_codes]

    for bucket in candidates:
        score = _placement_score(
            unit,
            bucket,
            role_targets,
            skill_targets.get(bucket["code"], {}),
            power_targets.get(bucket["code"], 0.0),
            relations,
            assigned_team_by_user,
        )
        if (
            chosen is None
            or score < best
            or (score == best and bucket["playerCount"] < chosen["playerCount"])
        ):
            chosen = bucket
            best = score
    if chosen is None:
        return buckets[team_codes[0]]
    return chosen


def _unit_set_skill(unit):
    size = max(_to_int(unit.get("size"), 1), 1)
    skills = unit.get("skillTotals", {}) or {}
    set_total = _to_float(skills.get("set"), 0.0)
    return set_total / float(size)


def _unit_attack_skill(unit):
    size = max(_to_int(unit.get("size"), 1), 1)
    skills = unit.get("skillTotals", {}) or {}
    attack_total = _to_float(skills.get("attack"), 0.0)
    return attack_total / float(size)


def _bucket_setter_quality(bucket):
    total = 0.0
    count = 0
    for unit in bucket.get("units", []):
        if unit.get("role") != "setter":
            continue
        total += _unit_set_skill(unit)
        count += 1
    if count <= 0:
        return 0.0
    return total / float(count)


def _bucket_attacker_load(bucket):
    total = 0.0
    for unit in bucket.get("units", []):
        if unit.get("role") == "attacker":
            total += _unit_attack_skill(unit)
    return total


def _bucket_central_load(bucket):
    total = 0.0
    for unit in bucket.get("units", []):
        if unit.get("role") != "central":
            continue
        skills = unit.get("skillTotals", {}) or {}
        size = max(_to_int(unit.get("size"), 1), 1)
        block_avg = _to_float(skills.get("block"), 0.0) / float(size)
        attack_avg = _to_float(skills.get("attack"), 0.0) / float(size)
        total += block_avg * 1.1 + attack_avg * 0.25
    return total


def _unit_prefer_to_users(unit, target_users, relations):
    if not target_users:
        return 0.0
    score = 0.0
    for player in unit.get("players", []):
        for edge in relations.get(player["userID"], []):
            if edge.get("type") != "prefer_together":
                continue
            if edge.get("other") in target_users:
                score += float(edge.get("weight", 0))
    return score


def _team_setter_user_ids(bucket):
    out = set()
    for unit in bucket.get("units", []):
        if unit.get("role") != "setter":
            continue
        for player in unit.get("players", []):
            out.add(player["userID"])
    return out


def _assign_unit_to_bucket(unit, bucket, assigned_team_by_user):
    bucket["units"].append(unit)
    bucket["playerCount"] += unit["size"]
    bucket["powerTotal"] += unit["powerTotal"]
    if unit["role"] in bucket["roleCounts"]:
        bucket["roleCounts"][unit["role"]] += 1
    for code in SKILL_CODES:
        bucket["skillTotals"][code] += unit["skillTotals"][code]
    for player in unit["players"]:
        assigned_team_by_user[player["userID"]] = bucket["code"]


def _bench_score(player, player_by_id):
    role = _player_role(player, player_by_id)
    skills = _player_skills(player)
    rating = _to_float(player.get("rating"), 5.0)
    return _player_power(role, skills) + rating * 0.85


def _lineup_slot_score(slot: int, player, player_by_id):
    role = _player_role(player, player_by_id)
    skills = _player_skills(player)
    rating = _to_float(player.get("rating"), 5.0)
    base = _player_power(role, skills) + rating * 0.55
    attack = skills["attack"]
    block = skills["block"]
    set_score = skills["set"]
    receive = skills["receive"]
    defense = skills["defense"]
    serve = skills["serve"]

    def role_bonus(target: str, bonus: float):
        if role == target:
            return bonus
        return 0.0

    if slot == 1:
        return base + serve * 1.5 + receive * 1.2 + defense * 0.8 + role_bonus("attacker", 1.1)
    if slot == 2:
        return base + set_score * 2.3 + serve * 0.6 + defense * 0.7 + role_bonus("setter", 4.5)
    if slot == 3:
        return base + block * 2.2 + attack * 1.1 + serve * 0.4 + role_bonus("central", 3.7)
    if slot == 4:
        return base + attack * 2.0 + serve * 1.0 + receive * 0.5 + role_bonus("attacker", 3.1)
    if slot == 5:
        return base + receive * 2.1 + defense * 2.0 + set_score * 0.4 + role_bonus("libero", 4.3)
    if slot == 6:
        return base + block * 1.2 + defense * 1.0 + receive * 0.9 + attack * 0.8 + role_bonus("central", 1.7)
    return base


def _order_team_players_for_lineup(players, player_by_id, lineup_size: int):
    if len(players) <= 1:
        return list(players)
    starters_limit = min(max(lineup_size, 1), len(SLOT_ORDER))
    available = list(players)
    ordered = []

    for slot in SLOT_ORDER:
        if len(ordered) >= starters_limit or not available:
            break
        best_idx = -1
        best_score = -math.inf
        for idx, player in enumerate(available):
            score = _lineup_slot_score(slot, player, player_by_id)
            if best_idx == -1 or score > best_score:
                best_idx = idx
                best_score = score
        if best_idx < 0:
            break
        ordered.append(available[best_idx])
        del available[best_idx]

    available.sort(key=lambda p: (-_bench_score(p, player_by_id), p["userID"]))
    return ordered + available


def _enforce_guest_owner_assignments(players, assignments):
    owner_by_guest = {}
    for player in players:
        user_id = _to_int(player.get("userID"), 0)
        owner_id = _to_int(player.get("guestOwnerID"), 0)
        if user_id < 0 and owner_id > 0:
            owner_by_guest[user_id] = owner_id
    if not owner_by_guest:
        return assignments

    by_user = {}
    for item in assignments:
        uid = _to_int(item.get("userID"), 0)
        if uid == 0:
            continue
        by_user[uid] = {
            "userID": uid,
            "team": (str(item.get("team")) if item.get("team") is not None else "").strip().upper(),
            "position": _to_int(item.get("position"), 0),
        }

    for guest_id, owner_id in owner_by_guest.items():
        owner = by_user.get(owner_id)
        if owner is None:
            continue
        guest = by_user.get(guest_id)
        if guest is None:
            guest = {
                "userID": guest_id,
                "team": owner["team"],
                "position": owner["position"] + 1,
            }
        guest["team"] = owner["team"]
        if guest["position"] < owner["position"]:
            guest["position"] = owner["position"] + 1
        by_user[guest_id] = guest
    return list(by_user.values())


def _normalize_assignments(assignments, team_codes):
    by_team = {code: [] for code in team_codes}
    by_team["unassigned"] = []
    allowed = set(team_codes)

    for item in assignments:
        uid = _to_int(item.get("userID"), 0)
        if uid == 0:
            continue
        team = (str(item.get("team")) if item.get("team") is not None else "").strip().upper()
        if team not in allowed:
            team = "unassigned"
        by_team[team].append(
            {
                "userID": uid,
                "team": team,
                "position": _to_int(item.get("position"), 0),
            }
        )

    out = []
    for code in list(team_codes) + ["unassigned"]:
        team_items = by_team.get(code, [])
        team_items.sort(key=lambda row: (row["position"], row["userID"]))
        for idx, row in enumerate(team_items):
            row["position"] = idx
            out.append(row)
    return out


def _validate_assignments(
    players,
    assignments,
    player_by_id,
    team_codes,
    capacity,
    role_targets,
):
    if not players or not assignments:
        return False

    allowed = set(team_codes)
    by_user = {}
    for item in assignments:
        uid = _to_int(item.get("userID"), 0)
        if uid == 0:
            continue
        team = (str(item.get("team")) if item.get("team") is not None else "").strip().upper()
        by_user[uid] = {
            "userID": uid,
            "team": team,
            "position": _to_int(item.get("position"), 0),
        }

    team_counts = {code: 0 for code in team_codes}
    role_counts = {
        code: {role: 0 for role in ROLE_TARGET_ORDER}
        for code in team_codes
    }

    for player in players:
        uid = player["userID"]
        item = by_user.get(uid)
        if item is None:
            return False
        team = item["team"]
        if team not in allowed:
            return False
        team_counts[team] += 1
        owner_id = _to_int(player.get("guestOwnerID"), 0)
        if owner_id > 0:
            owner = by_user.get(owner_id)
            if owner is not None and owner["team"] != team:
                return False
        role = _player_role(player, player_by_id)
        if role in role_counts[team]:
            role_counts[team][role] += 1

    for code in team_codes:
        if team_counts[code] > capacity.get(code, 0):
            return False
        for role in ROLE_TARGET_ORDER:
            target = role_targets.get(role, {}).get(code, 0)
            if role_counts[code][role] < target:
                return False
    return True


def split_teams(req: dict):
    players_in = req.get("players") or []
    relations_in = req.get("relations") or []

    norm_players = []
    for raw in players_in:
        user_id = _to_int(raw.get("userID"), 0)
        if user_id == 0:
            continue
        norm_players.append(
            {
                "userID": user_id,
                "guestOwnerID": _to_int(raw.get("guestOwnerID"), 0),
                "rating": _to_float(raw.get("rating"), 5.0),
                "playerType": _normalize_role(raw.get("playerType") or ""),
                "skills": _normalize_skills(raw.get("skills") or {}),
            }
        )

    n = len(norm_players)
    if n == 0:
        return {"assignments": []}

    team_codes = _sanitize_team_codes(req.get("teamCodes"), n)
    capacity = _sanitize_capacity(req.get("capacity"), n, team_codes)

    player_by_id = {p["userID"]: p for p in norm_players}
    relations = {}
    for raw in relations_in:
        a = _to_int(raw.get("userAID"), 0)
        b = _to_int(raw.get("userBID"), 0)
        if a == 0 or b == 0:
            continue
        rel_type = _normalize_relation_type(raw.get("relationType") or "")
        if not rel_type:
            continue
        weight = _to_int(raw.get("weight"), 0)
        if weight <= 0:
            continue
        if weight > 10:
            weight = 10
        relations.setdefault(a, []).append({"other": b, "type": rel_type, "weight": weight})
        relations.setdefault(b, []).append({"other": a, "type": rel_type, "weight": weight})

    units = _build_units(norm_players, player_by_id, relations)
    units.sort(
        key=lambda unit: (
            _role_priority(unit["role"]),
            -unit["relationWeight"],
            -unit["powerTotal"],
            -unit["size"],
            unit["id"],
        )
    )

    role_targets = _build_role_targets(units, team_codes, req.get("roleTargets"))
    skill_targets, power_targets = _build_stat_targets(units, team_codes, capacity)

    buckets = {}
    for code in team_codes:
        buckets[code] = {
            "code": code,
            "capacity": capacity.get(code, 0),
            "units": [],
            "playerCount": 0,
            "roleCounts": {"setter": 0, "libero": 0, "central": 0},
            "skillTotals": {skill: 0.0 for skill in SKILL_CODES},
            "powerTotal": 0.0,
        }

    assigned_team_by_user = {}
    remaining_units = list(units)

    # Stage 1: setters first by pass skill ("set"), not by overall rating.
    for code in team_codes:
        target = role_targets.get("setter", {}).get(code, 0)
        for _ in range(target):
            best_idx = -1
            best_set = -math.inf
            for idx, unit in enumerate(remaining_units):
                if unit.get("role") != "setter":
                    continue
                bucket = buckets[code]
                if bucket["playerCount"] + unit["size"] > bucket["capacity"]:
                    continue
                set_score = _unit_set_skill(unit)
                if best_idx == -1 or set_score > best_set:
                    best_idx = idx
                    best_set = set_score
            if best_idx < 0:
                break
            setter_unit = remaining_units.pop(best_idx)
            _assign_unit_to_bucket(setter_unit, buckets[code], assigned_team_by_user)

    # Stage 2: pull prefer_together links to already assigned setters.
    for code in team_codes:
        bucket = buckets[code]
        setter_users = _team_setter_user_ids(bucket)
        if not setter_users:
            continue
        while bucket["playerCount"] < bucket["capacity"]:
            best_idx = -1
            best_metric = -math.inf
            for idx, unit in enumerate(remaining_units):
                if bucket["playerCount"] + unit["size"] > bucket["capacity"]:
                    continue
                prefer = _unit_prefer_to_users(unit, setter_users, relations)
                if prefer <= 0:
                    continue
                base_penalty = _placement_score(
                    unit,
                    bucket,
                    role_targets,
                    skill_targets.get(code, {}),
                    power_targets.get(code, 0.0),
                    relations,
                    assigned_team_by_user,
                )
                # Strongly prioritize explicit preference links to setter core.
                metric = prefer * 100.0 - base_penalty
                if best_idx == -1 or metric > best_metric:
                    best_idx = idx
                    best_metric = metric
            if best_idx < 0:
                break
            linked_unit = remaining_units.pop(best_idx)
            _assign_unit_to_bucket(linked_unit, bucket, assigned_team_by_user)

    # Stage 3: strongest attackers go to teams with weaker setters.
    attacker_indices = [idx for idx, unit in enumerate(remaining_units) if unit.get("role") == "attacker"]
    attacker_indices.sort(key=lambda idx: (-_unit_attack_skill(remaining_units[idx]), remaining_units[idx]["id"]))
    teams_by_setter = sorted(team_codes, key=lambda code: (_bucket_setter_quality(buckets[code]), code))
    for code in teams_by_setter:
        chosen_pos = -1
        for pos, idx in enumerate(attacker_indices):
            unit = remaining_units[idx]
            bucket = buckets[code]
            if bucket["playerCount"] + unit["size"] <= bucket["capacity"]:
                chosen_pos = pos
                break
        if chosen_pos < 0:
            continue
        chosen_idx = attacker_indices.pop(chosen_pos)
        attacker_unit = remaining_units.pop(chosen_idx)
        _assign_unit_to_bucket(attacker_unit, buckets[code], assigned_team_by_user)
        attacker_indices = [i - 1 if i > chosen_idx else i for i in attacker_indices]

    # Stage 4: satisfy remaining mandatory role targets (libero/central + extra setter if needed).
    for role in ROLE_TARGET_ORDER:
        for code in team_codes:
            target = role_targets.get(role, {}).get(code, 0)
            while buckets[code]["roleCounts"].get(role, 0) < target:
                best_idx = -1
                best_score = math.inf
                for idx, unit in enumerate(remaining_units):
                    if unit["role"] != role:
                        continue
                    bucket = buckets[code]
                    if bucket["playerCount"] + unit["size"] > bucket["capacity"]:
                        continue
                    score = _placement_score(
                        unit,
                        bucket,
                        role_targets,
                        skill_targets.get(code, {}),
                        power_targets.get(code, 0.0),
                        relations,
                        assigned_team_by_user,
                    )
                    if best_idx == -1 or score < best_score:
                        best_idx = idx
                        best_score = score
                if best_idx < 0:
                    break
                forced_unit = remaining_units.pop(best_idx)
                _assign_unit_to_bucket(forced_unit, buckets[code], assigned_team_by_user)

    # Stage 5: weak-first balancing for the rest.
    remaining_units.sort(key=lambda unit: (unit["powerTotal"], unit["id"]))
    for unit in remaining_units:
        chosen = _choose_bucket_for_unit(
            unit,
            team_codes,
            buckets,
            role_targets,
            skill_targets,
            power_targets,
            relations,
            assigned_team_by_user,
        )
        _assign_unit_to_bucket(unit, chosen, assigned_team_by_user)

    lineup_size = _to_int(req.get("lineupSize"), 6)
    if lineup_size <= 0:
        lineup_size = 6

    assignments = []
    for code in team_codes:
        bucket_players = []
        for unit in buckets[code]["units"]:
            bucket_players.extend(unit["players"])
        ordered = _order_team_players_for_lineup(bucket_players, player_by_id, lineup_size)
        for pos, player in enumerate(ordered):
            assignments.append(
                {
                    "userID": player["userID"],
                    "team": code,
                    "position": pos,
                }
            )

    assignments = _enforce_guest_owner_assignments(norm_players, assignments)
    assignments = _normalize_assignments(assignments, team_codes)

    if not _validate_assignments(
        norm_players,
        assignments,
        player_by_id,
        team_codes,
        capacity,
        role_targets,
    ):
        # Return best-effort split; API caller has its own validator + local fallback.
        return {
            "assignments": assignments,
            "teamCodes": team_codes,
            "capacity": capacity,
            "roleTargets": role_targets,
            "valid": False,
        }

    return {
        "assignments": assignments,
        "teamCodes": team_codes,
        "capacity": capacity,
        "roleTargets": role_targets,
        "valid": True,
    }


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
