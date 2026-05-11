"""Script with all the necessary classes/functions to talk to the simod backend."""

import json
from loguru import logger
import os
import re
from typing import Any, Dict, Optional

from dotenv import load_dotenv
import requests
import urllib3

from ecotwin.common.system import System, get_env_file_path

# Disables insecure warning
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


class BackendConnector:
    """Class to handle the connecting/auth to the simod cluster handling backend."""

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

        if response.status_code != 200:
            logger.error(f"code: {response.status_code}")
            logger.info(response.content[:400])
            response.raise_for_status()

        # print(r.headers)
        # print(r.cookies)
        logger.info("Login passed")
        # token = r.json()['success']['token']
        # bearerToken = "Bearer "+token
        # self.session.headers.update({'Authorization' : bearerToken})
        # self.session.cookies['auth.strategy'] = 'local'
        # self.session.cookies['auth._refresh_token.local'] = 'false'
        # self.session.cookies['auth._token.local'] = bearerToken
        # r = self.session.post(self.url+'/api/login', data=login_data,
        #       verify=self.verify, proxies=self.proxies)
        # print(self.session.headers)
        # print(self.session.cookies)
        # logger.info("Test")
        # r = self.get('/admin/dashboard')
        # print(self.session.cookies)
        # print(r.cookies)

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
        # logger.debug("HEADER: {}".format(self.headers))
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
        # logger.debug("HEADER: {}".format(self.headers))
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
        # print(self.session.headers)
        # print(self.session.cookies)
        headers = {}  # self.headers.copy()
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
        # logger.debug("PUT: {}".format(self.url+api_name))
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
        # logger.debug("PUT: {}".format(self.url+api_name))
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
        proxies: Optional[Dict[str, str]] = None,
        headers: Optional[Dict[str, str]] = None,
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
            proxies {Optional[Dict[str, str]]} -- (default: None)
            headers {Optional[Dict]} -- (default: None)
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

    def getModelList(self):
        """
        Arguments:
            job_id {int} -- The id of the job to get the data of

        Returns:
            data {dict} -- The current set of the Job Data
        """
        r = requests.get(
            self.url + "/api/automated_user/form/list",
            headers=self.headers,
            verify=self.verify,
            proxies=self.proxies,
        )
        if r.status_code != 200:
            print(r.text)
            r.raise_for_status()

        return r.json()["success"]


class AjaxApiHandler:
    """Class to handle the connecting/authentication to the simod backend."""

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
        """Return the list of models."""

        response = self.api.getModelList()

        data = response
        models = []
        for item in data:
            name = item["name"]
            if "Automatisch" in name:
                models.append(name)

        return models


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
    credentials: Optional[Dict[str, str]] = None,
) -> Any:
    """Connect to the requested system and create a ApiHandler.

    Arguments:
        backend {System} -- The system to connect to.

    Returns:
        ApiHandler -- The connected ApiHandler.
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

    api_connector = BackendConnector(
        email=credentials["email"],
        password=credentials["passwd"],
        url=backend_url,
        verify=verify,
    )
    return AjaxApiHandler(api_connector)

    # api_connector = JobHandler(
    #     email=credentials["email"],
    #     password=credentials["passwd"],
    #     url=backend_url,
    #     verify=verify,
    # )

    # return ApiHandler(api_connector)
