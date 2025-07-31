# CS Onboarding Agent

An intelligent onboarding agent for OpenShift Cluster Service (OCM) that provides interactive guidance for new team members through automated workflows and Jira integration.

## Overview

The CS Onboarding Agent is designed to streamline the onboarding process for new engineers joining the OpenShift Cluster Service team. It provides:

- **Interactive Onboarding Workflows**: Step-by-step guidance through onboarding stages
- **Jira Integration**: Automatic ticket creation and tracking for onboarding tasks
- **Progress Tracking**: Real-time monitoring of onboarding completion
- **REST API**: Programmatic access to onboarding sessions and status
- **Contextual Help**: Smart assistance based on onboarding stage and user progress

## Features

### 🎯 Onboarding Workflow Management
- Multi-stage onboarding process (Welcome → Environment Setup → Team Introduction → First Tasks → Completion)
- Personalized guidance based on user role and experience
- Automatic progression tracking with completion validation

### 🎫 Jira Integration
- Automatic creation of onboarding tickets in OCM project
- Real-time status updates and progress tracking
- Integration with team workflows and sprint planning
- Support for custom fields and labels

### 🔄 Session Management
- Persistent onboarding sessions with state management
- Support for pausing and resuming onboarding
- Multi-user concurrent session handling
- Session cleanup and archival

### 📊 Progress Analytics
- Detailed progress reporting and metrics
- Completion time tracking and benchmarking
- Identification of common bottlenecks
- Team onboarding statistics

## Architecture

```
cs-onboarding-agent/
├── cmd/
│   └── onboarding-agent/
│       └── main.go              # Application entry point
├── pkg/
│   └── onboarding/
│       ├── agent.go            # Core onboarding logic
│       ├── jira.go             # Jira integration
│       └── service.go          # REST API service
└── README.md
```

### Core Components

- **OnboardingAgent**: Core business logic for managing onboarding workflows
- **OnboardingService**: REST API layer for external integrations
- **JiraClient**: Integration with Jira for ticket management
- **SessionManager**: Handles onboarding session lifecycle

## API Endpoints

### Start Onboarding Session
```http
POST /api/v1/onboarding/start
Content-Type: application/json

{
  "user_id": "user123",
  "username": "john.doe",
  "email": "john.doe@redhat.com"
}
```

### Process Message
```http
POST /api/v1/onboarding/message
Content-Type: application/json

{
  "session_id": "session123",
  "message": "I've completed setting up my development environment"
}
```

### Get Session Status
```http
GET /api/v1/onboarding/status/{session_id}
```

### List Sessions
```http
GET /api/v1/onboarding/sessions
```

### Health Check
```http
GET /api/v1/onboarding/health
```

## Configuration

### Environment Variables

The application requires the following environment variables:

```bash
# Jira Configuration
JIRA_PERSONAL_TOKEN=your_jira_personal_access_token

# Database Configuration (if using persistent storage)
DATABASE_URL=postgresql://user:password@localhost:5432/onboarding

# Service Configuration
PORT=8080
LOG_LEVEL=info
```

### Jira Configuration

The agent integrates with Red Hat Jira instance with the following default settings:
- **Base URL**: `https://issues.redhat.com`
- **Project Key**: `OCM`
- **Board ID**: `21633`

## Installation & Deployment

### Prerequisites
- Go 1.21+
- Access to Red Hat Jira instance
- Valid Jira Personal Access Token

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/crizzo71/AI-Experiements.git
   cd AI-Experiements/cs-onboarding-agent
   ```

2. **Set up environment variables**:
   ```bash
   export JIRA_PERSONAL_TOKEN="your_jira_token_here"
   ```

3. **Build and run**:
   ```bash
   go build -o onboarding-agent ./cmd/onboarding-agent
   ./onboarding-agent
   ```

### Docker Deployment

```dockerfile
# Example Dockerfile
FROM registry.access.redhat.com/ubi8/go-toolset:1.21 as builder
COPY . /workspace
WORKDIR /workspace
RUN go build -o onboarding-agent ./cmd/onboarding-agent

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /workspace/onboarding-agent /usr/local/bin/
EXPOSE 8080
CMD ["onboarding-agent"]
```

### OpenShift Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cs-onboarding-agent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cs-onboarding-agent
  template:
    metadata:
      labels:
        app: cs-onboarding-agent
    spec:
      containers:
      - name: onboarding-agent
        image: quay.io/your-org/cs-onboarding-agent:latest
        ports:
        - containerPort: 8080
        env:
        - name: JIRA_PERSONAL_TOKEN
          valueFrom:
            secretKeyRef:
              name: jira-credentials
              key: personal-token
---
apiVersion: v1
kind: Service
metadata:
  name: cs-onboarding-agent-service
spec:
  selector:
    app: cs-onboarding-agent
  ports:
  - port: 80
    targetPort: 8080
```

## Usage Examples

### Starting an Onboarding Session

```bash
curl -X POST http://localhost:8080/api/v1/onboarding/start \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "jdoe",
    "username": "john.doe",
    "email": "john.doe@redhat.com"
  }'
```

### Interacting with the Agent

```bash
curl -X POST http://localhost:8080/api/v1/onboarding/message \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "jdoe-1648123456",
    "message": "I need help setting up my development environment"
  }'
```

## Onboarding Stages

1. **Welcome**: Introduction and account setup verification
2. **Environment Setup**: Development environment configuration
3. **Team Introduction**: Meet the team and understand roles
4. **First Tasks**: Initial assignments and learning objectives
5. **Completion**: Final verification and feedback collection

## Integration with OCM Ecosystem

The CS Onboarding Agent integrates seamlessly with the OpenShift Cluster Manager ecosystem:

- **Authentication**: Uses OCM authentication middleware
- **Logging**: Integrated with OCM logging framework
- **Service Discovery**: Compatible with OCM service mesh
- **Monitoring**: Exports metrics for OCM monitoring stack

## Security Considerations

- All credentials stored as environment variables or Kubernetes secrets
- API endpoints protected with OCM authentication
- Session data encrypted at rest
- Audit logging for all onboarding activities
- Regular security scans and dependency updates

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Testing

```bash
# Run unit tests
go test ./pkg/...

# Run integration tests (requires environment setup)
go test -tags=integration ./test/...

# Run with coverage
go test -cover ./pkg/...
```

## Monitoring & Observability

The service exports the following metrics:
- Active onboarding sessions
- Completion rates by stage
- Average onboarding time
- Error rates and types

Health check endpoint provides:
- Service status
- Database connectivity
- External dependencies status

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../LICENSE) file for details.

## Support

For questions or support:
- Create an issue in this repository
- Contact the OCM team via Slack: `#ocm-team`
- Email: ocm-support@redhat.com

---

*Built with ❤️ for the OpenShift Cluster Service team*