from pathlib import Path


import yaml


class TwinInfo(object):
    def __init__(self, config):

        self.config = config


def read_twin_yaml(file_path: Path):
    """Read twin configuration from file."""
    with file_path.open("r") as stream:
        config = yaml.safe_load(stream)

    return TwinInfo(config)
