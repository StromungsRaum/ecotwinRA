import os
import yaml

from pathlib import Path

from ecotwin.common.api_connector import (
    get_backend_url,
)
from ecotwin.common.system import System, get_env_file_path


class TwinInfo(object):
    def __init__(self, config):

        self.config = config


def read_twin_yaml(file_path: Path):
    """Read twin configuration from file."""
    with file_path.open("r") as stream:
        config = yaml.safe_load(stream)

    return TwinInfo(config)


def save_dot_env_file(
    system: System,
    email: str,
    password: str,
) -> None:
    dot_env_file = get_env_file_path()
    dot_env_file.parent.mkdir(exist_ok=True)

    with dot_env_file.open("w") as env:
        env.write("# login data\n")
        env.write(f"EMAIL={email}\n")
        env.write(f"PASSWD={password}\n\n")
        for s in System:
            env.write(f"# {s.value.title()} system\n")
            system_upper = s.value.upper()
            url = get_backend_url(s)
            env.write(f"BACKEND_URL_{system_upper}={url}\n")
            email_system = f"EMAIL_{system_upper}"
            passwd_system = f"PASSWD_{system_upper}"
            if s == system:
                env.write(f"{email_system}=${{EMAIL}}\n")
                env.write(f"{passwd_system}=${{PASSWD}}\n")
            else:
                e = (
                    os.environ.get(email_system)
                    if email_system in os.environ
                    else "${EMAIL}"
                )
                env.write(f"{email_system}={e}\n")
                p = (
                    os.environ.get(passwd_system)
                    if passwd_system in os.environ
                    else "${PASSWD}"
                )
                env.write(f"{passwd_system}={p}\n")
            env.write("\n")
