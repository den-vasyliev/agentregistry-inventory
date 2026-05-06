/**
 * @jest-environment jsdom
 */

import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import { SubmitResourceDialog } from '../submit-resource-dialog'

// Mock toast
jest.mock('sonner', () => ({
  toast: {
    success: jest.fn(),
    error: jest.fn(),
  },
}))

// Mock window.open
const mockWindowOpen = jest.fn()
Object.defineProperty(window, 'open', {
  value: mockWindowOpen,
})

describe('SubmitResourceDialog', () => {
  const defaultProps = {
    open: true,
    onOpenChange: jest.fn(),
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders dialog when open', () => {
    render(<SubmitResourceDialog {...defaultProps} />)
    
    expect(screen.getByText('Submit Resource for Review')).toBeInTheDocument()
    expect(screen.getByText('Fill in the details to generate an Agent Registry manifest.')).toBeInTheDocument()
  })

  it('does not render when closed', () => {
    render(<SubmitResourceDialog {...defaultProps} open={false} />)
    
    expect(screen.queryByText('Submit Resource for Review')).not.toBeInTheDocument()
  })

  it('switches between resource types', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Default should be MCP Server
    expect(screen.getByRole('tab', { name: 'MCP Server' })).toHaveAttribute('data-state', 'active')
    
    // Click Agent tab
    await user.click(screen.getByRole('tab', { name: 'Agent' }))
    expect(screen.getByRole('tab', { name: 'Agent' })).toHaveAttribute('data-state', 'active')
    
    // Should show agent-specific fields
    expect(screen.getByLabelText(/Container Image/)).toBeInTheDocument()
  })

  it('validates required fields', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Try to proceed without filling required fields
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    // Should show validation error for name
    await waitFor(() => {
      expect(require('sonner').toast.error).toHaveBeenCalledWith('Name is required')
    })
  })

  it('generates correct MCP server manifest', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Fill in required fields for MCP server
    await user.type(screen.getByLabelText(/Name \*/), 'filesystem-tools')
    await user.type(screen.getByLabelText(/Version \*/), '1.0.0')
    await user.type(screen.getByLabelText(/Title/), 'Filesystem Tools')
    await user.type(screen.getByLabelText(/Description/), 'MCP server for file operations')
    await user.type(screen.getByLabelText(/Package Identifier/), 'ghcr.io/test/filesystem-tools:1.0.0')
    await user.type(screen.getByLabelText(/Repository URL/), 'https://github.com/test-org/filesystem-tools')
    
    // Proceed to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      expect(screen.getByText(/Generated Manifest/)).toBeInTheDocument()
    })
    
    // Check that manifest contains expected content
    const manifestText = screen.getByRole('code')
    expect(manifestText.textContent).toContain('apiVersion: agentregistry.dev/v1alpha1')
    expect(manifestText.textContent).toContain('kind: MCPServerCatalog')
    expect(manifestText.textContent).toContain('name: filesystem-tools')
    expect(manifestText.textContent).toContain('version: 1.0.0')
  })

  it('generates correct agent manifest', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Switch to Agent tab
    await user.click(screen.getByRole('tab', { name: 'Agent' }))
    
    // Fill in required fields for agent
    await user.type(screen.getByLabelText(/Name \*/), 'code-reviewer')
    await user.type(screen.getByLabelText(/Version \*/), '2.1.0')
    await user.type(screen.getByLabelText(/Container Image \*/), 'ghcr.io/test/code-reviewer:2.1.0')
    await user.selectOptions(screen.getByLabelText(/Framework/), 'langchain')
    
    // Proceed to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      const manifestText = screen.getByRole('code')
      expect(manifestText.textContent).toContain('kind: AgentCatalog')
      expect(manifestText.textContent).toContain('name: code-reviewer')
      expect(manifestText.textContent).toContain('framework: langchain')
    })
  })

  it('opens GitHub PR correctly', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Fill in required fields
    await user.type(screen.getByLabelText(/Name \*/), 'test-resource')
    await user.type(screen.getByLabelText(/Version \*/), '1.0.0')
    await user.type(screen.getByLabelText(/Repository URL/), 'https://github.com/test-org/test-repo')
    
    // Go to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      expect(screen.getByText(/Open PR on GitHub/)).toBeInTheDocument()
    })
    
    // Click Open PR
    await user.click(screen.getByText(/Open PR on GitHub/))
    
    expect(mockWindowOpen).toHaveBeenCalledWith(
      expect.stringContaining('https://github.com/test-org/test-repo/new/main?filename=resources/mcp-server/test-resource/test-resource-1.0.0.yaml'),
      '_blank'
    )
  })

  it('copies manifest to clipboard', async () => {
    const user = userEvent.setup()
    
    // Mock clipboard
    const mockClipboard = {
      writeText: jest.fn().mockResolvedValue(undefined),
    }
    Object.defineProperty(navigator, 'clipboard', {
      value: mockClipboard,
    })
    
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Fill in required fields and go to preview
    await user.type(screen.getByLabelText(/Name \*/), 'test-resource')
    await user.type(screen.getByLabelText(/Version \*/), '1.0.0')
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      expect(screen.getByText(/Copy/)).toBeInTheDocument()
    })
    
    // Click copy button
    await user.click(screen.getByText(/Copy/))
    
    expect(mockClipboard.writeText).toHaveBeenCalled()
    expect(require('sonner').toast.success).toHaveBeenCalledWith('Manifest copied to clipboard')
  })

  it('validates repository URL format', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Fill required fields with invalid repo URL
    await user.type(screen.getByLabelText(/Name \*/), 'test-resource')
    await user.type(screen.getByLabelText(/Version \*/), '1.0.0')
    await user.type(screen.getByLabelText(/Repository URL/), 'invalid-url')
    
    // Go to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      expect(screen.getByText(/Open PR on GitHub/)).toBeInTheDocument()
    })
    
    // Try to open PR with invalid URL
    await user.click(screen.getByText(/Open PR on GitHub/))
    
    expect(require('sonner').toast.error).toHaveBeenCalledWith('Please enter a valid GitHub repository URL')
  })

  it('handles skill resource type', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Switch to Skill tab
    await user.click(screen.getByRole('tab', { name: 'Skill' }))
    
    // Fill in skill-specific fields
    await user.type(screen.getByLabelText(/Name \*/), 'terraform-deploy')
    await user.type(screen.getByLabelText(/Version \*/), '1.5.0')
    await user.selectOptions(screen.getByLabelText(/Category/), 'infrastructure')
    
    // Proceed to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      const manifestText = screen.getByRole('code')
      expect(manifestText.textContent).toContain('kind: SkillCatalog')
      expect(manifestText.textContent).toContain('category: infrastructure')
    })
  })

  it('handles model resource type', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Switch to Model tab
    await user.click(screen.getByRole('tab', { name: 'Model' }))
    
    // Fill in model-specific fields
    await user.type(screen.getByLabelText(/Name \*/), 'claude-sonnet')
    await user.selectOptions(screen.getByLabelText(/Provider \*/), 'anthropic')
    await user.type(screen.getByLabelText(/Model/), 'claude-3-5-sonnet-20241022')
    
    // Proceed to manifest preview
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      const manifestText = screen.getByRole('code')
      expect(manifestText.textContent).toContain('kind: ModelCatalog')
      expect(manifestText.textContent).toContain('provider: anthropic')
    })
  })

  it('resets form when dialog closes', () => {
    const onOpenChange = jest.fn()
    const { rerender } = render(<SubmitResourceDialog open={true} onOpenChange={onOpenChange} />)
    
    // Fill some fields
    fireEvent.change(screen.getByLabelText(/Name \*/), { target: { value: 'test' } })
    
    // Close dialog
    rerender(<SubmitResourceDialog open={false} onOpenChange={onOpenChange} />)
    
    // Reopen dialog
    rerender(<SubmitResourceDialog open={true} onOpenChange={onOpenChange} />)
    
    // Form should be reset
    expect(screen.getByLabelText(/Name \*/).value).toBe('')
  })

  it('navigates back from manifest preview', async () => {
    const user = userEvent.setup()
    render(<SubmitResourceDialog {...defaultProps} />)
    
    // Fill required fields and go to preview
    await user.type(screen.getByLabelText(/Name \*/), 'test-resource')
    await user.type(screen.getByLabelText(/Version \*/), '1.0.0')
    await user.click(screen.getByText('Next: Preview Manifest'))
    
    await waitFor(() => {
      expect(screen.getByText(/Back/)).toBeInTheDocument()
    })
    
    // Click back
    await user.click(screen.getByText(/Back/))
    
    // Should be back to form
    expect(screen.getByText('Fill in the details to generate an Agent Registry manifest.')).toBeInTheDocument()
  })
})