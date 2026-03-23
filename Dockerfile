FROM python:3.12-slim-trixie
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

ARG GID=1000
ARG UID=1000
ARG BRANCH=int
ENV GID=${GID}
ENV UID=${UID}
ENV BRANCH=${BRANCH}

WORKDIR /app

# Install dependencies
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=uv.lock \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    uv sync --locked --no-install-project

COPY . /app

CMD ["uv", "run", "uvicorn", "ecotwin.api_server:app", "--host", "0.0.0.0", "--port", "8000"]
