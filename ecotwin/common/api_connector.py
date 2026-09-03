"""Script with all the necessary classes/functions to talk to the simod backend."""

import json
import os
import re
from typing import Any, Optional

import requests
import urllib3
from dotenv import load_dotenv
from loguru import logger

from ecotwin.common.system import System, get_env_file_path

# Disables insecure warning
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


class BackendConnector:
    """Class to handle the connecting/auth to the simod cluster handling backend.

    Legacy, session/CSRF-based connector kept working for the `AjaxApiHandler`
    (`/form/ajax`) integration path. New code should use `JobHandler` (token-based
    `/api/login`) instead - see `create_api_handler(..., legacy=True)`.
    """

    def __init__(
        self,
        email,
        password,
        url="https://backend.simod.de",
        verify=True,
        proxies=None,
    ):
        """Create the connection with the simod backend.

        It connects and authentificates and remembers the bearer token.

        Arguments:
            email {string} -- The Email of the cluster  account
            password {string} -- The password of the account

        Keyword Arguments:
            url {str} -- The backend to connect with.
                (default: {'https://backend.simod.de'})
            verify {bool} -- If true to verify the servers TLS certificate.
                (default: {True})

        Raises:
            r.raise_for_status: Raises the error message from the connection
                (Wrong Password, No Connection etc.).
        """
        login_data = {}
        login_data["email"] = email
        login_data["password"] = password

        if email is None:
            raise ValueError("Email is None")
        if password is None:
            raise ValueError("Password is None")
        if url is None:
            raise ValueError("URL is None")

        self.url = url
        self.headers = {}
        self.verify = verify
        self.proxies = proxies

        # login and get the token
        self.session = requests.sessions.Session()
        user_agent = (
            "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:40.0) Gecko/20100101 Firefox/40.1"
        )
        self.session.headers.update({"user-agent": user_agent})
        # self.session.auth = (email, password)

        login_page = self.session.get(
            self.url,
            verify=self.verify,
            proxies=self.proxies,
        )
        self.csrf_token = re.findall(
            r'<input type="hidden" name="_token" value="(.*)"', login_page.text
        )[0]
        # print(self.csrf_token)
        login_data["_token"] = self.csrf_token

        # self.session.headers['X-CSFR-TOKEN'] = csrf_token

        response = self.session.post(
            self.url + "/login",
            data=login_data,
            verify=self.verify,
            proxies=self.proxies,
        )
        logger.info(f"response: {response}")

        if response.status_code != 200:
            logger.error(f"code: {response.status_code}")
            logger.info(response.content[:400])
            response.raise_for_status()

        logger.info("Login passed")

    def get(
        self,
        api_path: str,
    ):
        """Send a get request to the connected backend.

        Arguments:
            api_path {str} -- Api path.

        Returns:
            _type_ -- The response.
        """
        return self.session.get(
            url=self.url + api_path,
            verify=self.verify,
            proxies=self.proxies,
            allow_redirects=False,
        )

    def get_json(
        self,
        api_path: str,
        data=None,
    ):
        """Send a get json request to the connected backend.

        Arguments:
            api_path {str} -- Api path.
            data {_type_} -- The json data to send with the request.

        Returns:
            _type_ -- The response.
        """
        return self.session.get(
            self.url + api_path,
            json=data,
            verify=self.verify,
            proxies=self.proxies,
            allow_redirects=False,
        )

    def post(
        self,
        api_path: str,
        data=None,
        content_type: Optional[str] = None,
    ):
        """Send a post request to the connected backend.

        Arguments:
            api_path {str} -- Api path.

        Keyword Arguments:
            data {_type_} -- The data to attach to the post request. (default: {None})
            content_type {str} -- The type of the content that is send. (default: {None})

        Returns:
            _type_ -- The response.
        """
        logger.debug(f"POST: {self.url + api_path}")
        headers = {}
        if content_type is not None:
            headers["Content-Type"] = content_type
        return self.session.post(
            self.url + api_path,
            data=data,
            headers=headers,
            verify=self.verify,
            proxies=self.proxies,
        )

    def post_json(
        self,
        api_path: str,
        data=None,
    ):
        """Send a post json request to the connected backend.

        Arguments:
            api_path {str} -- Api path.

        Keyword Arguments:
            data {_type_} -- The data to attach to the post request. (default: {None})

        Returns:
            _type_ -- The response.
        """
        return self.session.post(
            self.url + api_path,
            json=data,
            verify=self.verify,
            proxies=self.proxies,
            allow_redirects=False,
        )

    def put(
        self,
        api_path: str,
    ):
        """Send a put request to the connected backend.

        Arguments:
            api_path {str} -- Api path.

        Returns:
            _type_ -- The response.
        """
        return self.session.put(
            self.url + api_path,
            verify=self.verify,
            proxies=self.proxies,
            allow_redirects=False,
        )

    def put_json(
        self,
        api_path: str,
        data,
    ):
        """Send a put json request to the connected backend.

        Arguments:
            api_path {str} -- Api path.
            data {_type_} -- The json data.

        Returns:
            _type_ -- The response.
        """
        return self.session.put(
            self.url + api_path,
            json=data,
            verify=self.verify,
            proxies=self.proxies,
            allow_redirects=False,
        )


class JobHandler:
    """Class to handle the connecting/authentification to the simod cluster handling backend."""

    def __init__(
        self,
        email: str,
        password: str,
        url: str = "https://backend.simod.de",
        verify: bool = True,
        proxies: Optional[dict[str, str]] = None,
        headers: Optional[dict[str, str]] = None,
        timeout: int = 500,
        bearer_token: Optional[str] = None,
    ) -> None:
        """Create the connection with the simod backend.

        Arguments:
            email {string} -- The Email of the cluster account
            password {string} -- The password of the account

        Keyword Arguments:
            url {str} -- The backend to connect with (default: {'https://backend.simod.de'})
            verify {bool} -- Optional. A Boolean or a String indication to verify
                the servers TLS certificate or not. (default: {True})
            proxies {Optional[dict[str, str]]} -- (default: None)
            headers {Optional[dict]} -- (default: None)
            bearer_token {Optiona[str]} -- If none, login an retrive a the token (default: None)

        Raises:
            r.raise_for_status: Raises the error message from the connection
                (Wrong Password, No Connection etc.).
        """
        login_data = {
            "email": email,
            "password": password,
        }
        self.url = url
        self.headers = {}
        self.timeout = timeout
        if headers is not None:
            self.headers.update(headers)
        self.verify = verify
        self.proxies = proxies

        from requests import Session
        from requests.adapters import HTTPAdapter
        from urllib3.util import Retry

        self.session = Session()
        retries = Retry(
            connect=10,
            total=10,
            backoff_factor=0.2,
        )
        adapter = HTTPAdapter(max_retries=retries)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)

        if bearer_token is None:
            # login and get the token
            req_return = self.session.post(
                url=f"{self.url}/api/login",
                data=login_data,
                verify=self.verify,
                proxies=self.proxies,
                headers=self.headers,
                timeout=self.timeout,
            )

            if req_return.status_code != 200:
                raise req_return.raise_for_status()

            bearer_token = req_return.json()["success"]["token"]

        self.bearer_token = bearer_token
        self.headers["Authorization"] = "Bearer " + self.bearer_token

    def get_campaigns(self):
        """Return the submission campaigns (product/application templates) usable by this account."""
        r = requests.get(
            self.url + "/api/automated_user/submission/campaigns",
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
        )
        if r.status_code != 200:
            print(r.text)
            r.raise_for_status()

        return r.json()["success"]

    def get_route(self, route: str) -> dict:
        """Return entities."""
        r = requests.get(
            self.url + f"/{route}",
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
        )
        if r.status_code != 200:
            logger.error(f"{route}: {r.text}")
            r.raise_for_status()

        json_data = json.loads(r.text)

        return json_data

    def get_entity(self, entity: str) -> dict:
        """Return entities."""
        return self.get_route(f"api/platform/{entity}")

    def get_single_entity(self, entity: str, entity_id: Any) -> dict:
        """Return entities."""
        r = requests.get(
            self.url + f"/api/platform/{entity}/{str(entity_id)}",
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
        )
        if r.status_code != 200:
            logger.error(f"{entity}: {r.text}")
            r.raise_for_status()

        json_data = json.loads(r.text)

        return json_data

    def submit(self, payload: dict):
        """Submit geometry/process/material/reporter parameters for a campaign.

        Builds the digital twin and creates the simulation in one atomic call
        against the `/api/automated_user/submission` endpoint.

        Arguments:
            payload {dict} -- Matches the endpoint's request shape: campaign_var,
                geometry, process_parameters, material_selections, reporter_parameters.

        Returns:
            dict -- The `success` payload (simulation_id, digital_twin_id, status).
        """
        r = requests.post(
            self.url + "/api/automated_user/submission",
            json=payload,
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
            timeout=self.timeout,
        )
        if r.status_code != 200:
            print(r.text)
            r.raise_for_status()

        return r.json()["success"]

    def post_file(self, file_path: str) -> Any:
        """Upload a file to the backend via `Api\\FileController` and return its id.

        Arguments:
            file_path {str} -- Path to the file to upload.

        Returns:
            Any -- The id of the created `BigFileAsset`, for later reference
                (e.g. as a component parameter value).
        """
        with open(file_path, "rb") as fh:
            r = requests.post(
                self.url + "/api/files",
                files={"image": (os.path.basename(file_path), fh)},
                headers=self.headers,
                verify=self.verify,
                proxies=self.proxies,
                timeout=self.timeout,
            )

        if r.status_code != 200:
            logger.error(f"post_file: {r.text}")
            r.raise_for_status()

        logger.info(f"post_file: {r.text}")

        return r.json()["id"]

    def create_component(
        self,
        name: str,
        component_type: str,
        parameters: Optional[dict[str, Any]] = None,
        components: Optional[dict[str, Any]] = None,
    ) -> Any:
        """Create a Component via `Api\\Platform\\ComponentController` and return its id.

        Arguments:
            name {str} -- Component name.
            component_type {str} -- uuid of the component type
                (see `get_entity("component_types")` for the available types
                and the parameter/child-slot schema each one expects).

        Keyword Arguments:
            parameters {Optional[dict]} -- Type-dependent parameter values
                (strings, numbers, or a file id from `post_file()` for a
                CAD-file parameter). (default: None)
            components {Optional[dict]} -- Child component references, keyed
                by slot. (default: None)

        Returns:
            Any -- The id of the created component.
        """
        r = requests.post(
            self.url + "/api/platform/components",
            json={
                "name": name,
                "component_type": component_type,
                "parameters": parameters or {},
                "components": components or {},
            },
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
            timeout=self.timeout,
        )
        if r.status_code != 200:
            logger.error(f"create_component: {r.text}")
            r.raise_for_status()

        return r.json()["id"]


class AjaxApiHandler:
    """Class to handle the connecting/authentication to the simod backend.

    Legacy path: scrapes the staff-only `/form/ajax` admin DataTables endpoint via
    `BackendConnector`'s session/CSRF login. Superseded by `ApiHandler`
    (token-based, hits the proper `/api/automated_user/submission/*` API), but
    kept working - see `create_api_handler(..., legacy=True)`.
    """

    def __init__(self, api_connector):
        """Create the backend handler."""
        self.api = api_connector

    def get_models(
        self,
    ):
        """Return the list of models."""

        response = self.api.get("/form/ajax")

        if response.status_code != 200:
            if response.status_code == 302:
                logger.error("Tries to redirect back to login")
                raise ValueError("Bad login credentials.")
            logger.info(response.content[:400])
            raise response.raise_for_status()
        response = json.loads(response.content)
        data = response["data"]

        models = set()
        for item in data:
            name_en = item["name_en"]
            name = item["name"]
            if "Automatisch" in name:
                bits = name_en.split("|")
                if len(bits) > 1:
                    models.add(bits[1])

        return list(models)


class ApiHandler:
    """Class to handle the connecting/authentication to the simod backend."""

    def __init__(self, api_connector):
        """Create the backend handler."""
        self.api = api_connector

    def get_models(
        self,
    ):
        """Return the list of submission campaigns (product/application templates) usable by this account."""
        return self.api.get_campaigns()

    def get(self, route: str) -> dict:
        return self.api.get_route(route)

    def get_entity(self, entity: str) -> dict:
        return self.api.get_entity(entity)

    def get_single_entity(self, entity: str, entity_id: Any) -> dict:
        return self.api.get_single_entity(entity, entity_id)

    def submit(self, payload: dict):
        """Submit a full model (geometry + digital twin + simulation) in one call."""
        return self.api.submit(payload)

    def post_file(self, file_path: str) -> Any:
        return self.api.post_file(file_path)

    def create_component(
        self,
        name: str,
        component_type: str,
        parameters: Optional[dict[str, Any]] = None,
        components: Optional[dict[str, Any]] = None,
    ) -> Any:
        return self.api.create_component(name, component_type, parameters, components)


def _get_credentials_from_env(backend: System):
    """Retrieve the credentials for the backend of the desired system."""
    email_system_key = f"EMAIL_{backend.upper()}"
    if email_system_key not in os.environ and "EMAIL" not in os.environ:
        raise ValueError(
            f"Couldn't retrieve credentials from env as EMAIL or {email_system_key} is missing."
        )
    passwd_system_key = f"PASSWD_{backend.upper()}"
    if passwd_system_key not in os.environ and "PASSWD" not in os.environ:
        raise ValueError(
            "Couldn't retrieve credentials from env as PASSWD or PASSWD_<system> is missing."
        )
    return {
        "email": (
            os.environ[email_system_key]
            if email_system_key in os.environ
            else os.environ["EMAIL"]
        ),
        "passwd": (
            os.environ[passwd_system_key]
            if passwd_system_key in os.environ
            else os.environ["PASSWD"]
        ),
    }


def get_backend_url(backend: System):
    """Retrieve the backend url based of the desired system."""
    system_backend_url_env_key = f"BACKEND_URL_{backend.upper()}"
    if system_backend_url_env_key in os.environ:
        return os.environ[system_backend_url_env_key]

    systems = {
        "prod": "backend.simod.de",
        "int": "backend.int.simod.de",
        "test": "backend.test.simod.de",
        "dev": "backend.dev.simod.de",
    }

    return systems[backend]


def create_api_handler(
    system: System,
    credentials: Optional[dict[str, str]] = None,
    legacy: bool = False,
) -> Any:
    """Connect to the requested system and create an API handler.

    Arguments:
        backend {System} -- The system to connect to.

    Keyword Arguments:
        legacy {bool} -- If True, use the old session/CSRF `/form/ajax` scraping
            path (`AjaxApiHandler`) instead of the token-based
            `/api/automated_user/submission/*` API (`ApiHandler`, default).
            (default: {False})

    Returns:
        ApiHandler | AjaxApiHandler -- The connected handler.
    """
    logger.info(f"Connecting to backend: {system}")

    dot_env_path = get_env_file_path()
    if dot_env_path is not None:
        logger.info(f"Using credentials {dot_env_path}")
        load_dotenv(dot_env_path, interpolate=True)

    credentials = _get_credentials_from_env(backend=system)
    logger.info(f"credentials: {credentials}")

    backend_url = get_backend_url(backend=system)

    verify = system != "dev"

    if legacy:
        api_connector = BackendConnector(
            email=credentials["email"],
            password=credentials["passwd"],
            url=backend_url,
            verify=verify,
        )
        return AjaxApiHandler(api_connector)

    job_handler = JobHandler(
        email=credentials["email"],
        password=credentials["passwd"],
        url=backend_url,
        verify=verify,
    )

    return ApiHandler(job_handler)
