# BACKEND STANDARDS (MANDATORY)

## 1. Directory Structure & Layers
- **Route**: Only for endpoint definitions.
- **Handler**: Input validation & HTTP responses.
- **Service**: Business logic & 3rd party/external service calls.
- **Repository**: SQL Query strings only.
- **Database**: Execution of queries (must reside in /database folder).

## 2. Coding Rules
- **Security**: Use PARAMETERIZED QUERIES only. No string formatting for SQL.
- **Comments**: Max 5 words. Only for critical logic.
- **API Grouping**: Group by prefix in a single file (e.g., /v1/bus).
- **Execution**: Repository must call functions from the Database folder to execute queries.

## 3. Workflow
Before generating code, check if the logic belongs in Service or Repository. 
If it's an external API call, it MUST be in Service.