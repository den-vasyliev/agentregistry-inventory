#!/usr/bin/env python3
"""
Simple MCP client to test Agent Registry MCP tools
"""
import json
import requests

MCP_URL = "http://localhost:8083"

def call_mcp_tool(tool_name, arguments=None):
    """Call an MCP tool via SSE transport"""
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": arguments or {}
        }
    }

    print(f"\n{'='*60}")
    print(f"Calling tool: {tool_name}")
    print(f"Arguments: {json.dumps(arguments, indent=2)}")
    print(f"{'='*60}\n")

    response = requests.post(
        f"{MCP_URL}/message",
        json=payload,
        headers={"Content-Type": "application/json"}
    )

    if response.status_code == 200:
        result = response.json()
        print("✓ Success!")
        print(json.dumps(result, indent=2))
        return result
    else:
        print(f"✗ Error: {response.status_code}")
        print(response.text)
        return None

def list_tools():
    """List available MCP tools"""
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list",
        "params": {}
    }

    print(f"\n{'='*60}")
    print("Listing available MCP tools")
    print(f"{'='*60}\n")

    response = requests.post(
        f"{MCP_URL}/message",
        json=payload,
        headers={"Content-Type": "application/json"}
    )

    if response.status_code == 200:
        result = response.json()
        if "result" in result and "tools" in result["result"]:
            tools = result["result"]["tools"]
            print(f"Found {len(tools)} tools:")
            for tool in tools:
                if "master" in tool["name"] or "event" in tool["name"]:
                    print(f"  - {tool['name']}: {tool.get('description', 'No description')}")
        return result
    else:
        print(f"✗ Error: {response.status_code}")
        print(response.text)
        return None

if __name__ == "__main__":
    print("Agent Registry MCP Tool Tester")
    print("=" * 60)

    # 1. List tools to see master agent tools
    list_tools()

    # 2. Get master agent status
    call_mcp_tool("get_master_agent_status")

    # 3. Emit a test event
    call_mcp_tool("emit_event", {
        "type": "test-event",
        "message": "Testing MCP integration - simulated pod crash",
        "severity": "warning",
        "source": "mcp-test-client"
    })

    # Wait a moment for processing
    print("\n⏳ Waiting 3 seconds for master agent to process event...")
    import time
    time.sleep(3)

    # 4. Get status again to see changes
    call_mcp_tool("get_master_agent_status")

    # 5. Get recent events
    call_mcp_tool("get_recent_events", {"limit": 10})

    print("\n" + "="*60)
    print("✓ Test complete!")
    print("="*60)
