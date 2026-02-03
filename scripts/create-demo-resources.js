#!/usr/bin/env node

const API_URL = process.env.API_URL || 'http://localhost:8080';

async function createDemoServer() {
  console.log('📦 Creating demo MCP Server...');

  const server = {
    name: 'demo-weather-server',
    version: '1.0.0',
    title: 'Demo Weather MCP Server',
    description: 'A demonstration MCP server that provides real-time weather information and forecasts for any location worldwide. Features include current conditions, 7-day forecasts, severe weather alerts, and historical weather data.',
    repository: {
      url: 'https://github.com/demo-org/weather-server',
      source: 'github'
    },
    websiteUrl: 'https://weather-demo.example.com',
    packages: [
      {
        identifier: '@demo/weather-server',
        version: '1.0.0',
        registryType: 'npm',
        runtimeHint: 'node',
        runtimeArguments: [
          {
            name: 'main',
            value: 'index.js',
            description: 'Main entry point',
            required: true
          }
        ],
        environmentVariables: [
          {
            name: 'WEATHER_API_KEY',
            description: 'API key for weather service',
            required: true
          },
          {
            name: 'CACHE_TTL',
            description: 'Cache time-to-live in seconds',
            value: '3600',
            required: false
          },
          {
            name: 'LOG_LEVEL',
            description: 'Logging level (debug, info, warn, error)',
            value: 'info',
            required: false
          }
        ],
        transport: {
          type: 'stdio'
        }
      }
    ]
  };

  const response = await fetch(`${API_URL}/admin/v0/servers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(server)
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to create server: ${error}`);
  }

  console.log('✅ Server created successfully');
  return await response.json();
}

async function createDemoSkill() {
  console.log('⚡ Creating demo Skill...');

  const skill = {
    name: 'demo-data-analysis',
    version: '1.0.0',
    title: 'Advanced Data Analysis',
    description: 'A comprehensive skill for analyzing datasets, generating statistical insights, and creating beautiful visualizations. Supports CSV, JSON, and Excel formats. Features include descriptive statistics, correlation analysis, trend detection, and automated report generation.',
    repository: {
      url: 'https://github.com/demo-org/data-analysis-skill',
      source: 'github'
    }
  };

  const response = await fetch(`${API_URL}/admin/v0/skills`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(skill)
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to create skill: ${error}`);
  }

  console.log('✅ Skill created successfully');
  return await response.json();
}

async function createDemoAgent() {
  console.log('🤖 Creating demo Agent...');

  const agent = {
    name: 'demo-customer-support',
    version: '1.0.0',
    title: 'Customer Support Agent',
    description: 'An intelligent customer support agent powered by Claude that handles inquiries, provides solutions, troubleshoots issues, and escalates complex cases when needed. Trained on comprehensive product knowledge and best practices for customer service. Features multi-turn conversation support, context retention, and sentiment analysis.',
    image: 'demo-org/customer-support-agent:1.0.0',
    language: 'python',
    framework: 'langgraph',
    modelProvider: 'anthropic',
    modelName: 'claude-sonnet-4.5',
    repository: {
      url: 'https://github.com/demo-org/customer-support-agent',
      source: 'github'
    },
    packages: [
      {
        registryType: 'docker',
        identifier: 'demo-org/customer-support-agent',
        version: '1.0.0',
        transport: {
          type: 'http'
        }
      }
    ]
  };

  const response = await fetch(`${API_URL}/admin/v0/agents`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(agent)
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to create agent: ${error}`);
  }

  console.log('✅ Agent created successfully');
  return await response.json();
}

async function createDemoModel() {
  console.log('🧠 Creating demo Model...');

  const model = {
    name: 'demo-sentiment-analyzer',
    provider: 'huggingface',
    model: 'distilbert-base-uncased-finetuned-sst-2',
    baseUrl: 'https://api-inference.huggingface.co/models',
    description: 'A state-of-the-art sentiment analysis model fine-tuned on customer feedback data. Classifies text as positive, negative, or neutral with high accuracy (94.7%). Optimized for real-time inference with low latency (45ms average). Supports multiple languages and batch processing. Ideal for analyzing customer reviews, social media posts, and support tickets.'
  };

  const response = await fetch(`${API_URL}/admin/v0/models`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(model)
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to create model: ${error}`);
  }

  console.log('✅ Model created successfully');
  return await response.json();
}

async function publishResource(type, name, version) {
  console.log(`📢 Publishing ${type}: ${name}@${version}...`);

  const endpoint = type === 'model'
    ? `${API_URL}/admin/v0/models/${encodeURIComponent(name)}/publish`
    : `${API_URL}/admin/v0/${type}s/${encodeURIComponent(name)}/versions/${encodeURIComponent(version)}/publish`;

  const response = await fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' }
  });

  if (!response.ok) {
    const error = await response.text();
    console.warn(`⚠️  Failed to publish ${type}: ${error}`);
    return false;
  }

  console.log(`✅ ${type} published successfully`);
  return true;
}

async function deployResource(type, name, version) {
  if (type !== 'server' && type !== 'agent') {
    console.log(`ℹ️  Skipping deployment for ${type} (only servers and agents can be deployed)`);
    return;
  }

  console.log(`🚀 Deploying ${type}: ${name}@${version} to Kubernetes...`);

  const resourceType = type === 'server' ? 'mcp' : 'agent';

  const deployment = {
    serverName: name,
    version: version,
    resourceType: resourceType,
    runtime: 'kubernetes',
    config: {
      namespace: 'demo-registry',
      replicas: 1
    },
    preferRemote: false
  };

  const response = await fetch(`${API_URL}/admin/v0/deployments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(deployment)
  });

  if (!response.ok) {
    const error = await response.text();
    console.warn(`⚠️  Failed to deploy ${type}: ${error}`);
    return false;
  }

  console.log(`✅ ${type} deployed successfully`);
  return true;
}

async function main() {
  console.log('🌱 Creating demo resources for AgentRegistry\n');
  console.log(`📍 API URL: ${API_URL}\n`);

  try {
    // Create resources
    const server = await createDemoServer();
    console.log('');

    const skill = await createDemoSkill();
    console.log('');

    const agent = await createDemoAgent();
    console.log('');

    const model = await createDemoModel();
    console.log('');

    // Wait a bit for resources to be indexed
    console.log('⏳ Waiting 2 seconds for resources to be indexed...\n');
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Publish resources
    await publishResource('server', 'demo-weather-server', '1.0.0');
    await publishResource('skill', 'demo-data-analysis', '1.0.0');
    await publishResource('agent', 'demo-customer-support', '1.0.0');
    await publishResource('model', 'demo-sentiment-analyzer', 'latest');
    console.log('');

    // Wait a bit before deploying
    console.log('⏳ Waiting 2 seconds before deploying...\n');
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Deploy server and agent to cluster
    await deployResource('server', 'demo-weather-server', '1.0.0');
    await deployResource('agent', 'demo-customer-support', '1.0.0');
    console.log('');

    console.log('✅ All demo resources created successfully!\n');
    console.log('📋 Summary:');
    console.log('  • MCP Server: demo-weather-server (published & deployed)');
    console.log('  • Skill: demo-data-analysis (published)');
    console.log('  • Agent: demo-customer-support (published & deployed)');
    console.log('  • Model: demo-sentiment-analyzer (published)');
    console.log('\n🌐 View in UI: http://localhost:3000');

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    process.exit(1);
  }
}

main();
