import typer

from loguru import logger

from ecotwin.common.api_connector import (
    create_api_handler,
)
from ecotwin.common.system import System


def models_command(
    system: System = typer.Option(
        System.INT,
        "--system",
        help="System",
    ),
) -> None:
    """Get information about StrömungsRaum models."""
    logger.info("Get information about models")

    api_handler = create_api_handler(system)

    models = api_handler.get_models()

    logger.info("Models:")
    for model in models:
        logger.info(f"\t{model}")
