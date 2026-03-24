import os

from enum import Enum
from pathlib import Path
from loguru import logger


class System(str, Enum):
    """Contains the different simod systems available."""

    PROD = "prod"
    TEST = "test"
    INT = "int"
    DEV = "dev"


def is_valid_system(system: str) -> bool:
    """Check if the given system string is a valid system."""
    try:
        System(system)
        return True
    except ValueError:
        return False


def get_env_file_path() -> Path | None:

    if "HOME" not in os.environ:
        logger.warning("HOME environment variable is not set")
        return None

    user_folder = Path(os.environ["HOME"])
    dot_env_file = user_folder / ".config" / "ecotwin" / ".env"
    if dot_env_file.exists():
        return dot_env_file

    return None
