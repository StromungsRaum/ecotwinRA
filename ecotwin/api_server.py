import json

from pathlib import Path
from typing import Any

from fastapi import FastAPI
from pydantic import BaseModel
from loguru import logger

from fastapi.responses import FileResponse

from ecotwin.common.api_connector import (
    create_api_handler,
)
from ecotwin.common.system import System

app = FastAPI()


favicon_path = Path(__file__).parents[1] / "assets" / "favicon.ico"


@app.get("/favicon.ico", include_in_schema=False)
async def favicon():
    return FileResponse(str(favicon_path.absolute()))


@app.get("/")
async def root() -> dict[str, str]:
    return {"message": "EcoTwin"}


@app.get("/models")
async def models() -> dict[str, str]:

    system: System = System.DEV

    api_handler = create_api_handler(system)

    models = api_handler.get_models()

    return {"models": f"{models}"}


def common_get(entity: str) -> dict[str, str]:
    system: System = System.DEV

    handler = create_api_handler(system)

    entities = handler.get_entity(entity)
    pretty = json.dumps(entities, indent=4)

    logger.info(f"{entity}:\n{pretty}")

    return entities


@app.get("/components")
async def components() -> dict[str, str]:
    components = common_get("components")
    return {"components": str(components)}


@app.get("/digital_twins")
async def digital_twins() -> dict[str, str]:
    digital_twins = common_get("digital_twins")
    return {"digital_twins": str(digital_twins)}


@app.get("/materials")
async def materials() -> dict[str, str]:
    materials = common_get("materials")
    return {"materials": str(materials)}


@app.get("/simulations")
async def simulations() -> dict[str, str]:
    simulations = common_get("simulations")
    return {"simulations": str(simulations)}


class SubmissionPayload(BaseModel):
    campaign_var: str
    name: str | None = None
    geometry: dict[str, Any]
    process_parameters: list[dict[str, Any]] = []
    material_selections: dict[str, str] = {}
    reporter_parameters: list[dict[str, Any]] = []
    project_id: int | None = None


@app.post("/submit")
async def submit(payload: SubmissionPayload) -> dict[str, Any]:
    system: System = System.INT

    api_handler = create_api_handler(system)

    return api_handler.submit(payload.model_dump(exclude_none=True))
