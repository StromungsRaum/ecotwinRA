import typer
from typing import Optional

from loguru import logger
from dotenv import load_dotenv

from ecotwin.common.system import System, get_env_file_path
from ecotwin.common.twin_config import (
    save_dot_env_file,
)


app = typer.Typer(help="Login to StrömungsRaum", pretty_exceptions_enable=False)


@app.callback(invoke_without_command=False)
def login_callback(ctx: typer.Context) -> None:
    """Login to the EcoTwin system."""
    pass


@app.command("store")
def store_command(
    system: Optional[System] = typer.Option(
        None,
        "--system",
        help="System",
    ),
    email: Optional[str] = typer.Option(
        None,
        "--email",
        envvar=["SR_USERNAME", "USERNAME"],
        help="User email",
    ),
    password: Optional[str] = typer.Option(
        None,
        "--pass",
        envvar=["SR_PASSWORD", "PASSWORD"],
        help="User password",
    ),
) -> None:
    """Store login information for the Twin system."""
    if system:
        logger.info(f"Using system: {system.value}")

    dot_env_path = get_env_file_path()
    if dot_env_path.exists():
        load_dotenv(dot_env_path, interpolate=True)

    save_dot_env_file(system, email, password)
