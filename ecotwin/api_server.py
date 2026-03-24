from pathlib import Path

from fastapi import FastAPI
from fastapi.responses import FileResponse

from ecotwin.common.api_connector import (
    create_api_connector,
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

    system: System = System.INT

    backend_handler = create_api_connector(system)

    models = backend_handler.get_models()

    return {"models": f"{models}"}
