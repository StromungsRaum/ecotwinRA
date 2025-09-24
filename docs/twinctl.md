# ECOTWIN Controller **twinctl**

To run **twinctl** install python 3.11 or higher and install [**uv**](https://docs.astral.sh/uv/).
Make sure uv is in the PATH or invoke it directly.

The _twinctl_ can invoked directly.
This method uses
[inline script metadata](https://packaging.python.org/en/latest/specifications/inline-script-metadata/#inline-script-metadata)
and _uv_ to create a temporary virtual environment to execute the script.

```shell
bin/twinctl
```

Alternatively a virtual environment can be created and then used to launch the script.
This is done using the following steps:

```shell
cd ecotwinRA
uv sync
source .venv/bin/activate
twinctl
```
