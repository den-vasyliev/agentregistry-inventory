import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateString: string): string {
  try {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return dateString
  }
}

export function formatDateTime(dateString: string): string {
  try {
    return new Date(dateString).toLocaleString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateString
  }
}

export function getStatusBadgeStyles(status: string): string {
  switch (status) {
    case "Running":
      return 'bg-green-500/10 text-green-600 border-green-500/20'
    case "Failed":
      return 'bg-red-500/10 text-red-600 border-red-500/20'
    case "External":
      return 'bg-teal-500/10 text-teal-600 border-teal-500/20'
    default:
      return 'bg-gray-500/10 text-gray-600 border-gray-500/20'
  }
}

export function getStatusColor(status: string): string {
  switch (status) {
    case 'active':
      return 'bg-green-600'
    case 'deprecated':
      return 'bg-yellow-600'
    case 'deleted':
      return 'bg-red-600'
    default:
      return 'bg-gray-600'
  }
}
