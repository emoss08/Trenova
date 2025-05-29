<div align="center">
<h1 align="center"><b>Trenova Suite</b></h1>
  
  **Revolutionizing Transportation Management Through AI Innovation**
  
  [![Go Report Card](https://goreportcard.com/badge/github.com/emoss08/trenova)](https://goreportcard.com/report/github.com/emoss08/trenova)
  [![Discord](https://dcbadge.limes.pink/api/server/https://discord.gg/XDBqyvrryq?style=flat-square&theme=default-inverted)](https://discord.gg/XDBqyvrryq)
</div>

> [!IMPORTANT]
> Trenova is currently in development and is not yet suitable for production use.
> We are actively working on building the core functionality and will be releasing it in the future.

## Table of Contents

- [Vision](#vision)
- [Industry Challenges & Our Mission](#industry-challenges--our-mission)
- [Why Open Source?](#why-open-source)
- [Architecture & Technology Stack](#architecture--technology-stack)
- [Quick Start](#quick-start)
- [Documentation](#documentation)
- [System Requirements (Self-Hosting)](#system-requirements-self-hosting)
- [Database Support](#database-support)
- [Support & Community](#support--community)
- [Commercial Support](#commercial-support)
- [Contribution Policy](#contribution-policy)
- [License](#license)

## Vision

Trenova emerged from witnessing firsthand the daily struggles of Over-The-Road carriers in the United States. Our vision extends beyond just creating another TMS - we're building an ecosystem that empowers transportation companies to thrive in an increasingly complex industry.

### The Current State of TMS

Traditional Transportation Management Systems (TMS) have created significant barriers:

- **Financial Burden**: Enterprise TMS solutions often cost $50,000+ annually, making them inaccessible to small and mid-sized carriers who make up 90% of the industry
- **Technical Debt**: Legacy systems built on outdated technology stacks require expensive maintenance and limit innovation
- **Operational Inefficiency**: Users spend 30-40% of their time on repetitive tasks that could be automated
- **Integration Challenges**: Most systems operate in silos, forcing manual data entry and increasing error rates
- **Complex Implementation**: Average implementation times of 6-12 months delay time-to-value
- **Limited Innovation**: Closed systems prevent carriers from adapting to changing market conditions
- **Data Silos**: Valuable operational data remains trapped in disparate systems, hindering decision-making
- **Compliance Burden**: Manual tracking of FMCSA regulations increases risk and consumes resources

### Trenova's Revolutionary Approach

We're transforming transportation management through:

1. **AI-First Design**: Rather than bolting on AI features, we've built intelligence into every aspect of the system
2. **Workflow Automation**: Identifying and automating repetitive tasks to free up human capital
3. **Real-Time Intelligence**: Providing actionable insights when they matter most
4. **Adaptive Learning**: Systems that learn and improve from user interactions
5. **Collaborative Innovation**: Building features based on real user needs and feedback

We believe transportation companies deserve better. Trenova represents a fundamental shift in how carriers manage their operations, combining artificial intelligence with human expertise to create a system that truly serves its users.

## Industry Challenges & Our Mission

The transportation industry faces unprecedented challenges that require innovative solutions.

### Critical Industry Pain Points

1. **Operational Inefficiencies**
   - 40% of fleet capacity underutilized due to poor planning
   - Average dispatcher manages only 15-20 trucks manually
   - 25% of drive time lost to suboptimal routing
   - 20% of revenue lost to inefficient load matching

2. **Compliance and Safety**
   - $5,000+ average cost per DOT audit
   - 70% of carriers struggle with HOS compliance
   - Manual log tracking leads to frequent violations
   - Delayed maintenance increases safety risks

3. **Technology Barriers**
   - Legacy systems unable to adapt to new regulations
   - Limited integration capabilities with modern tools
   - High training and onboarding costs
   - Poor mobile accessibility for drivers

4. **Data Management**
   - Critical business data spread across multiple systems
   - Limited real-time visibility into operations
   - Difficult to track KPIs and performance metrics
   - Manual reporting consumes valuable time

### Our Mission

Trenova's mission is to revolutionize transportation management through:

1. **Accessibility**
   - Making enterprise-grade TMS technology available to all carriers
   - Reducing implementation time from months to days
   - Providing intuitive interfaces that minimize training needs
   - Offering flexible deployment options

2. **Intelligent Automation**
   - Automating 80% of routine dispatcher tasks
   - Real-time optimization of fleet operations
   - Predictive maintenance scheduling
   - Smart compliance monitoring and alerts

3. **Data Empowerment**
   - Centralizing operational data in one platform
   - Providing actionable business intelligence
   - Enabling data-driven decision making
   - Real-time performance monitoring

4. **Industry Innovation**
   - Creating open standards for transportation technology
   - Enabling rapid adaptation to market changes
   - Building an ecosystem of integrated solutions

## Why Open Source?

Trenova's commitment to open source goes beyond just sharing code - it's about creating a new paradigm for transportation technology.

### The Power of Open Source in Transportation

1. **Innovation Through Collaboration**
   - Transportation challenges are inherently interconnected
   - Solutions require input from dispatchers, drivers, maintenance staff, and management
   - Open source ensures transparency and allows the community to provide real-world feedback, though all development is handled internally.
   - Cross-pollination of ideas from different sectors enhances feature development

2. **Transparency and Trust**
   - Carriers can audit security and compliance features
   - Full visibility into data handling and privacy measures
   - Clear understanding of system capabilities and limitations
   - No vendor lock-in or hidden functionalities

3. **Community-Informed Evolution**
   - Features prioritized based on community feedback and user needs
   - Rapid bug fixes and security updates driven by community reports
   - Shared knowledge base for common challenges

4. **Unlimited Customization**
   - Carriers can modify any aspect of the system
   - Custom integrations with existing tools
   - Industry-specific workflow adaptations
   - Regional compliance requirement implementations

5. **Democratizing TMS Technology**
   - Eliminating financial barriers to entry
   - Enabling small carriers to compete effectively
   - Fostering industry-wide innovation
   - Creating a level playing field

### Our Open Source Commitment

- **Regular Code Releases**: Maintaining a predictable release schedule
- **Security First**: Regular security audits and responsible disclosure policy
- **Plugin Architecture**: Enabling easy extension of core functionality
- **Transparent Development**: Open roadmap and transparent decision-making process

## Architecture & Technology Stack

Trenova is built on a modern, scalable microservices architecture:

### Core Technologies

- **Backend**: Go (Golang) - chosen for its performance, simplicity, and strong typing
- **Frontend**: React with TypeScript - providing a responsive and intuitive user interface
- **Database**: PostgreSQL - ensuring data reliability and advanced querying capabilities
- **Cache**: Redis - for session management and high-performance caching
- **Infrastructure**: Docker & Docker Compose - enabling easy deployment and scaling

### Microservices Architecture

- **Main API**: Core business logic and data management
- **Email Service**: Dedicated microservice for email operations and template management
- **Workflow Service**: Powered by Hatchet for complex workflow orchestration
- **File Storage**: MinIO for object storage and document management
- **Message Queue**: RabbitMQ for asynchronous communication between services

### Supporting Infrastructure

- **Reverse Proxy**: Caddy for automatic HTTPS and load balancing
- **Connection Pooling**: PGBouncer for database connection optimization
- **Monitoring**: Built-in health checks and metrics collection

## Quick Start

### Prerequisites

- Docker 20.10+ with Docker Compose V2
- 12GB+ RAM and 6+ CPU cores allocated to Docker
- 30GB+ available disk space

### Local Development

```bash
# Clone the repository
git clone https://github.com/emoss08/trenova.git
cd trenova

# Start the development environment
docker-compose -f docker-compose-local.yml up -d

# Access the application
# Frontend: http://localhost:5173
# API: http://localhost:3001
```

### Production Deployment

```bash
# Clone the repository
git clone https://github.com/emoss08/trenova.git
cd trenova

# Configure environment variables
cp config/config.example.yaml config/config.yaml
# Edit config.yaml with your settings

# Deploy with production configuration
docker-compose -f docker-compose-prod.yml up -d

# Access the application
# Application: https://your-domain.com (via Caddy)
# Workflow Dashboard: http://your-server:8080
```

For detailed deployment instructions and system requirements, see the [System Requirements](SYSTEM_REQUIREMENTS.md) documentation.

## Documentation

### Deployment & Administration

- **[System Requirements](SYSTEM_REQUIREMENTS.md)** - Comprehensive resource requirements, scaling guidelines, and deployment instructions
- [Getting Started Guide](docs/development/setup.md) - Development environment setup
- [Production Deployment](docs/deployment/) - Production deployment strategies

### Development Resources

- [Development Guidelines](docs/development/guidelines.md) - Coding standards and best practices
- [API Reference](docs/api/openapi.yaml) - Complete API documentation
- [User Documentation](docs/user-guide.md) - End-user application guide

### Architecture Documentation

- [Microservices Architecture](microservices/) - Individual service documentation
- [Email Service](microservices/email/README.md) - Email handling and templates
- [Workflow Service](microservices/workflow/) - Hatchet-based workflow engine

## System Requirements (Self-Hosting)

> **Note**: These requirements are only applicable if you plan to **self-host** Trenova. For our **[Managed Hosting Services](#managed-hosting-services)**, all infrastructure requirements are handled by us.

Trenova is a comprehensive microservices-based application with specific resource requirements for optimal self-hosted performance.

### Self-Hosting Resource Overview

- **Minimum**: 6 cores, 12GB RAM, 30GB storage
- **Production (1-50 users)**: 12+ cores, 24GB+ RAM, 100GB+ SSD
- **Production (50-200 users)**: 20+ cores, 48GB+ RAM, 200GB+ SSD
- **Production (200+ users)**: 32+ cores, 96GB+ RAM, 500GB+ SSD

📋 **[View Complete Self-Hosting Requirements](SYSTEM_REQUIREMENTS.md)** - Detailed resource allocation, scaling guidelines, network requirements, and deployment checklists for self-hosted installations.

## Database Support

| Database     | Status             | Notes                    |
|-------------|--------------------|-----------------------------|
| PostgreSQL  | ✅ Full Support    | Recommended               |
| MySQL       | ⭕ Not Supported   | Not Planned                |
| SQLite      | ⚠️ Limited         | Development only          |

## Support & Community

We believe in building a strong, supportive community around Trenova:

### Community Resources

- **[GitHub Issues](https://github.com/emoss08/trenova/issues)** - Bug reports and feature requests
- **[GitHub Discussions](https://github.com/emoss08/trenova/discussions)** - Community discussions, questions, and support
- **[Discord Server](https://discord.gg/XDBqyvrryq)** - Real-time community chat and support
- **[Documentation](#documentation)** - Comprehensive guides and references

### Getting Help

1. **Check Documentation**: Start with our [System Requirements](SYSTEM_REQUIREMENTS.md) and [Documentation](#documentation)
2. **Search Issues**: Look through existing [GitHub Issues](https://github.com/emoss08/trenova/issues) for similar problems
3. **Community Discussion**: Ask questions in [GitHub Discussions](https://github.com/emoss08/trenova/discussions)
4. **Discord Support**: Join our [Discord server](https://discord.gg/XDBqyvrryq) for real-time help
5. **Report Bugs**: Create a detailed issue on [GitHub Issues](https://github.com/emoss08/trenova/issues)


## Commercial Support

For organizations requiring additional support, we offer enterprise-grade services:

### Managed Hosting Services

- **Fully Managed Deployment** - We host, maintain, and manage your Trenova instance
- **Automatic Updates & Upgrades** - Seamless application updates with zero downtime
- **Enterprise Infrastructure** - High-availability, scalable cloud infrastructure
- **Backup & Disaster Recovery** - Automated backups with guaranteed recovery times
- **Security & Compliance** - Enterprise-grade security with compliance certifications
- **Performance Monitoring** - 24/7 system monitoring with proactive optimization

### Support Services

- **Priority Issue Resolution** - Fast-track bug fixes and technical support
- **Custom Feature Development** - Tailored functionality for your business needs
- **Deployment Assistance** - Expert help with self-hosted production deployments
- **Training & Consultation** - Comprehensive staff training and best practices
- **System Optimization** - Performance tuning and scaling guidance
- **Migration Services** - Seamless migration from existing TMS solutions

### Enterprise Features

- **Dedicated Support Team** - Direct access to Trenova developers
- **SLA Guarantees** - Contractual uptime and response time commitments
- **Custom Integrations** - Connect with your existing systems and workflows
- **White-label Solutions** - Branded deployments for your organization
- **Compliance Support** - Help meeting industry-specific regulations
- **Multi-tenant Architecture** - Secure isolation for enterprise deployments

### Contact Information

- **Email**: <support@trenova.app>
- **Business Inquiries**: <sales@trenova.app>
- **Security Issues**: <security@trenova.app>

*Response times: Community support (best effort), Commercial support (guaranteed SLA)*

## Contribution Policy

> ⚠️ **Code Contributions Not Accepted**  
> While Trenova is open source for transparency and community insight, we are **not accepting external code contributions** at this time. Our development team is focused on building the core functionality and ensuring a cohesive vision for the platform.

### How You Can Help

While we don't accept code contributions, you can still contribute to the project:

- **🐛 Report Issues**: Help us identify bugs and improvement opportunities through [GitHub Issues](https://github.com/emoss08/trenova/issues)
- **💡 Feature Requests**: Share ideas for new functionality and enhancements
- **📝 Documentation**: Suggest improvements to guides, documentation, and system requirements
- **🤝 Community Support**: Help other users in [GitHub Discussions](https://github.com/emoss08/trenova/discussions) and [Discord](https://discord.gg/XDBqyvrryq)
- **🔍 Testing**: Report compatibility issues and deployment experiences

### Community Feedback

We actively welcome and value:
- Feedback on features and usability
- Bug reports with detailed reproduction steps
- Suggestions for system improvements
- Real-world usage experiences and case studies

Thank you for your understanding and continued support as we work towards making Trenova the future of transportation management.

## License

Trenova is distributed under the FSL-1.1-ALv2 License. See [LICENSE](LICENSE) file for details.
