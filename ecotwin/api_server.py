"""HTTP API for running EcoTwin CLI commands.

This module provides a minimal FastAPI application that forwards requests to the
existing `twinctl` click-based CLI.

The goal is to enable HTTP-based automation of the same workflows exposed by the
command line tool.

Example:
  curl -X POST http://localhost:8000/run -H "Content-Type: application/json" \
    -d '{"args": ["info"]}'

Run:
  uvicorn ecotwin.api_server:app --reload
"""

from __future__ import annotations

import io
import sys
from contextlib import contextmanager, redirect_stderr, redirect_stdout
from typing import Optional, Sequence


@contextmanager
def _redirect_stdin(new_stdin):
    """Context manager to temporarily replace sys.stdin."""
    old_stdin = sys.stdin
    sys.stdin = new_stdin
    try:
        yield
    finally:
        sys.stdin = old_stdin


try:
    from fastapi import FastAPI, HTTPException
    from pydantic import BaseModel
except ImportError as e:  # pragma: no cover
    FastAPI = None  # type: ignore[assignment]
    HTTPException = RuntimeError  # type: ignore[assignment]

    class BaseModel:
        """Minimal BaseModel fallback for environments without pydantic."""

        def __init__(self, **data):
            for key, value in data.items():
                setattr(self, key, value)

        def dict(self):
            return self.__dict__

    _missing_fastapi = e  # type: ignore[name-defined]

from ecotwin.twinctl import twinctl


class TwinctlRequest(BaseModel):
    """Request body for `/run`.

    `args` are the positional/flag arguments passed to the CLI (e.g. `["info"]`).
    "file" is an optional `--file` flag that selects a YAML config file.
    "input" is an optional stdin payload passed to the CLI.
    """

    args: Sequence[str] = []
    file: Optional[str] = None
    input: Optional[str] = None


class TwinctlResult(BaseModel):
    exit_code: int
    output: str


app = None  # type: ignore[assignment]

if FastAPI is not None:
    app = FastAPI(
        title="EcoTwin Twin Control API",
        description="Expose twinctl commands over HTTP.",
        version="0.1.0",
    )

    @app.get("/health")
    def health() -> dict[str, str]:
        """Simple liveness endpoint."""

        return {"status": "ok"}


def _invoke_twinctl(cli_args: list[str], input: Optional[str] = None) -> TwinctlResult:
    """Invoke `twinctl` directly, capturing output without using CliRunner."""

    stdout_buffer = io.StringIO()
    stderr_buffer = io.StringIO()

    stdin_buffer = io.StringIO(input or "")

    exit_code = 0
    exc: Optional[Exception] = None

    with (
        redirect_stdout(stdout_buffer),
        redirect_stderr(stderr_buffer),
        _redirect_stdin(stdin_buffer),
    ):
        try:
            # `standalone_mode=False` prevents click from calling sys.exit
            twinctl.main(args=cli_args, standalone_mode=False)
        except SystemExit as e:
            # click uses SystemExit to signal exit codes
            exit_code = int(e.code) if isinstance(e.code, int) else 1
        except Exception as e:  # pragma: no cover
            exc = e
            exit_code = 1

    output = "".join([stdout_buffer.getvalue(), stderr_buffer.getvalue()])
    result = TwinctlResult(exit_code=exit_code, output=output)

    if exc:
        raise exc

    return result


if FastAPI is not None:

    @app.post("/run", response_model=TwinctlResult)
    def run(request: TwinctlRequest) -> TwinctlResult:
        """Run the `twinctl` CLI with the provided arguments."""

        cli_args: list[str] = []
        if request.file:
            cli_args += ["--file", request.file]
        cli_args += list(request.args)

        try:
            result = _invoke_twinctl(cli_args, input=request.input)
        except Exception as exc:  # pragma: no cover
            raise HTTPException(
                status_code=500,
                detail={
                    "exit_code": 1,
                    "output": str(exc),
                    "exception": repr(exc),
                },
            )

        if result.exit_code != 0:
            raise HTTPException(
                status_code=500,
                detail={
                    "exit_code": result.exit_code,
                    "output": result.output,
                },
            )

        return result

    @app.get("/info/system", response_model=TwinctlResult)
    def info_system(
        file: Optional[str] = None,
        output_format: Optional[str] = None,
        root_name: Optional[str] = None,
        output_dir: Optional[str] = None,
    ) -> TwinctlResult:
        """Run `twinctl info system` and return the captured output."""

        cli_args: list[str] = ["info", "system"]
        if file:
            cli_args += ["--file", file]
        if output_format:
            cli_args += ["--format", output_format]
        if root_name:
            cli_args += ["--root-name", root_name]
        if output_dir:
            cli_args += ["--output-dir", output_dir]

        result = _invoke_twinctl(cli_args)
        if result.exit_code != 0:
            raise HTTPException(status_code=500, detail=result.dict())
        return result

    @app.get("/info/models", response_model=TwinctlResult)
    def info_models(
        file: Optional[str] = None,
        system: Optional[str] = None,
    ) -> TwinctlResult:
        """Run `twinctl info models` and return the captured output."""

        cli_args: list[str] = ["info", "models"]
        if file:
            cli_args += ["--file", file]
        if system:
            cli_args += ["--system", system]

        result = _invoke_twinctl(cli_args)
        if result.exit_code != 0:
            raise HTTPException(status_code=500, detail=result.dict())
        return result

    class LoginStoreRequest(BaseModel):
        file: Optional[str] = None
        system: Optional[str] = None
        email: Optional[str] = None
        password: Optional[str] = None

    @app.post("/login/store", response_model=TwinctlResult)
    def login_store(request: LoginStoreRequest) -> TwinctlResult:
        """Run `twinctl login store` and return the captured output."""

        cli_args: list[str] = ["login", "store"]
        if request.file:
            cli_args += ["--file", request.file]
        if request.system:
            cli_args += ["--system", request.system]
        if request.email:
            cli_args += ["--email", request.email]
        if request.password:
            cli_args += ["--pass", request.password]

        result = _invoke_twinctl(cli_args)
        if result.exit_code != 0:
            raise HTTPException(status_code=500, detail=result.dict())
        return result

else:  # pragma: no cover

    def run(*args, **kwargs):
        raise RuntimeError(
            "fastapi is not installed; install fastapi and pydantic to enable the HTTP API"
        )

    def info_system(*args, **kwargs):
        raise RuntimeError(
            "fastapi is not installed; install fastapi and pydantic to enable the HTTP API"
        )

    def info_models(*args, **kwargs):
        raise RuntimeError(
            "fastapi is not installed; install fastapi and pydantic to enable the HTTP API"
        )

    def login_store(*args, **kwargs):
        raise RuntimeError(
            "fastapi is not installed; install fastapi and pydantic to enable the HTTP API"
        )


def main() -> None:
    """Run the API server via `python -m ecotwin.api_server`."""

    if FastAPI is None or app is None:
        raise RuntimeError(
            "FastAPI or Pydantic is not installed. Install dependencies (fastapi, pydantic, uvicorn) "
            "to run the HTTP API server."
        )

    import uvicorn

    uvicorn.run("ecotwin.api_server:app", host="0.0.0.0", port=8000, log_level="info")


if __name__ == "__main__":
    main()
