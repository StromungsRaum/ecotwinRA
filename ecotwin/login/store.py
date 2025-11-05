import click

from loguru import logger
from dotenv import load_dotenv

from ecotwin.common.system import System, get_env_file_path
from ecotwin.common.twin_config import (
    save_dot_env_file,
)


@click.command(help="Store login information for the Twin system")
@click.option(
    "--system",
    "system",
    type=click.Choice(System, case_sensitive=False),  # type: ignore
    help=("System"),
)
@click.option(
    "--email",
    "email",
    type=click.STRING,
    envvar=["SR_USERNAME", "USERNAME"],
    help=("User email"),
)
@click.option(
    "--pass",
    "password",
    type=click.STRING,
    envvar=["SR_PASSWORD", "PASSWORD"],
    help=("User password"),
)
@click.pass_obj
def store(twin_info, system, email, password):
    logger.info(f"Using system: {system.value}")

    dot_env_path = get_env_file_path()
    if dot_env_path.exists():
        load_dotenv(dot_env_path, interpolate=True)

    save_dot_env_file(system, email, password)
