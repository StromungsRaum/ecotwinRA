import json

from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException
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


def single_get(entity: str, entity_id: Any) -> dict[str, str]:
    system: System = System.DEV

    handler = create_api_handler(system)

    entities = handler.get_single_entity(entity, entity_id)
    pretty = json.dumps(entities, indent=4)

    logger.info(f"{entity}:\n{pretty}")

    return entities


@app.get("/files")
async def files() -> dict[str, str]:
    system: System = System.DEV

    handler = create_api_handler(system)

    files = handler.get("api/files")
    pretty = json.dumps(files, indent=4)

    logger.info(f"{files}:\n{pretty}")

    return {"files": str(files)}


@app.get("/test_file_upload")
async def test_file_upload() -> dict[str, str]:
    system: System = System.DEV

    handler = create_api_handler(system)

    file = "/home/sava/data/pump/int_5088/preprocessing/zj_200_65_impeller.step"

    result = handler.post_file(file)

    return {"result": str(result)}


@app.get("/components")
async def components() -> dict[str, str]:
    components = common_get("components")
    return {"components": str(components)}


@app.get("/components/{component_id}")
async def single_component(component_id: Any) -> dict[str, str]:
    component = single_get("components", component_id)
    return {"component": str(component)}


class ComponentPayload(BaseModel):
    name: str
    component_type: str
    parameters: dict[str, Any] = {}
    components: dict[str, Any] = {}
    # Path to a CAD file to attach, and which parameter slot to attach it under
    # (see get_entity("component_types") for the parameter names a given type
    # accepts). Example of the post_file() -> create_component() flow: the file
    # is uploaded first, and the id it comes back with is what the component
    # parameter actually stores - not the file itself.
    file_path: str | None = None
    file_parameter: str | None = None


@app.post("/components")
async def create_component(payload: ComponentPayload) -> dict[str, Any]:
    system: System = System.DEV

    handler = create_api_handler(system)

    parameters = dict(payload.parameters)

    if payload.file_path is not None:
        if payload.file_parameter is None:
            raise HTTPException(
                400, "file_parameter is required when file_path is set"
            )
        file_id = handler.post_file(payload.file_path)
        parameters[payload.file_parameter] = file_id

    component_id = handler.create_component(
        name=payload.name,
        component_type=payload.component_type,
        parameters=parameters,
        components=payload.components,
    )

    return {"component_id": component_id}


@app.get("/digital_twins")
async def digital_twins() -> dict[str, str]:
    digital_twins = common_get("digital_twins")
    return {"digital_twins": str(digital_twins)}


@app.get("/digital_twins/{id}")
async def single_digital_twin(id: Any) -> dict[str, str]:
    dt = single_get("digital_twins", id)
    return {"digital_twin": str(dt)}


@app.get("/materials")
async def materials() -> dict[str, str]:
    materials = common_get("materials")
    return {"materials": str(materials)}


@app.get("/materials/{id}")
async def single_material(id: Any) -> dict[str, str]:
    mat = single_get("materials", id)
    return {"material": str(mat)}


@app.get("/simulations")
async def simulations() -> dict[str, str]:
    simulations = common_get("simulations")
    return {"simulations": str(simulations)}


@app.get("/simulations/{id}")
async def single_simulation(id: Any) -> dict[str, str]:
    sim = single_get("simulations", id)
    return {"simulation": str(sim)}


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
