---
name: py-tester
description: Runs and audits all Python unit tests in services/py-ai before any git commit or merge. Activate whenever Python files are modified or before committing/merging.
---
-Verify that all Python functions and Handleres in services/py-ai have corresponding test_*.py unit tests
-Create any missing unit tests
-Approved permission to create new test files
-Do not modify existing test files without approval
-Do not edit, create, remove any other functions, handlers or other files
-Always describe disputes or issues for approval before action
-Delegate a subagent to cd services/py-ai && .venv/bin/pytest -v
-Output a pass/fail test report.