"""Explicit, idempotent demo data for this project's local Docker database."""
from pathlib import Path
import subprocess


def main() -> None:
    root = Path(__file__).resolve().parent.parent
    sql = (root / "db" / "seeds" / "local-demo.sql").read_bytes()
    subprocess.run(
        [
            "docker", "compose", "-f", str(root / "docker-compose.yml"),
            "exec", "-T", "db", "sh", "-c",
            'exec psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"',
        ],
        input=sql,
        cwd=root,
        check=True,
    )


if __name__ == "__main__":
    main()
