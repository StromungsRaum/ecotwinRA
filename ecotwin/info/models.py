import typer

from loguru import logger

from ecotwin.common.api_connector import (
    create_api_connector,
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

    backend_handler = create_api_connector(system)

    models = backend_handler.get_models()

    logger.info("Models:")
    for model in models:
        logger.info(f"\t{model[0]} - {model[1]}")
