import json
from pathlib import Path

import typer
from loguru import logger

from ecotwin.common.api_connector import create_api_handler
from ecotwin.common.system import System


def submit_command(
    campaign_var: str = typer.Option(
        ...,
        "--campaign-var",
        help="The submission campaign to submit against (see `info models` for available values)",
    ),
    params_file: Path = typer.Option(
        ...,
        "--params-file",
        help="JSON file with geometry/process_parameters/material_selections/reporter_parameters",
        exists=True,
        readable=True,
    ),
    system: System = typer.Option(
        System.INT,
        "--system",
        help="System",
    ),
) -> None:
    """Submit a full model (geometry + digital twin + simulation) to StrömungsRaum.

    Exercises the `/api/automated_user/submission` endpoint independently of
    Platform Mesh/kcp - the standalone smoke-test client for that API.
    """
    payload = json.loads(params_file.read_text())
    payload["campaign_var"] = campaign_var

    api_handler = create_api_handler(system)

    result = api_handler.submit(payload)

    logger.info(f"Submitted: {result}")
