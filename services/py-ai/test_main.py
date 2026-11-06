from fastapi.testclient import TestClient
from main import app, get_python_service_health_status

client = TestClient(app)


def test_get_python_service_health_status_function_returns_expected_dictionary():
    response_dictionary = get_python_service_health_status()
    assert response_dictionary == {"status": "ok", "service": "pi-ai"}


def test_health_endpoint_returns_http_status_code_two_hundred():
    http_response = client.get("/health")
    assert http_response.status_code == 200


def test_health_endpoint_returns_expected_json_payload():
    http_response = client.get("/health")
    assert http_response.json() == {"status": "ok", "service": "pi-ai"}
