import click

from loguru import logger

from ecotwin.common.api_connector import (
    create_api_connector,
)
from ecotwin.common.system import System


@click.command(help="Get information about StrömungsRaum models")
@click.option(
    "--system",
    "system",
    type=click.Choice(System, case_sensitive=False),  # type: ignore
    default=System.INT,
    help=("System"),
)
@click.pass_obj
def command(twin_info, system):
    logger.info("Get information about models")

    backend_handler = create_api_connector(system)

    models = backend_handler.get_models()

    logger.info(f"Models: {models}")
