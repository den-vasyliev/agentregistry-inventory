# Governance Controller — Integration Guide

This document describes how an external governance controller should update catalog resources with trust verification and governance scores.

## CRD Status Field

All four catalog resources expose the same status subresource path:

```
.status.publisher
```

### Fields

| Field | Type | Description |
|---|---|---|
| `verifiedPublisher` | bool | Publisher identity verified |
| `verifiedOrganization` | bool | Organization verified |
| `score` | int (0–100) | Numeric governance score |
| `grade` | string (`A`/`B`/`C`/`D`/`F`) | Letter grade derived from score |
| `gradedAt` | RFC3339 timestamp | When score was last set |

### Supported Resources

| Resource | API Group/Version |
|---|---|
| `mcpservercatalogs` | `agentregistry.dev/v1alpha1` |
| `agentcatalogs` | `agentregistry.dev/v1alpha1` |
| `skillcatalogs` | `agentregistry.dev/v1alpha1` |
| `modelcatalogs` | `agentregistry.dev/v1alpha1` |

## Patch Example

Use the `status` subresource to avoid overwriting spec fields:

```bash
kubectl patch mcpservercatalog <name> \
  --subresource=status \
  --type=merge \
  -p '{
    "status": {
      "publisher": {
        "verifiedPublisher": true,
        "verifiedOrganization": true,
        "score": 87,
        "grade": "B",
        "gradedAt": "2026-02-19T10:00:00Z"
      }
    }
  }'
```

Same pattern applies for `agentcatalog`, `skillcatalog`, and `modelcatalog`.

## Grade Thresholds

| Grade | Score Range |
|---|---|
| A | 90–100 |
| B | 80–89 |
| C | 70–79 |
| D | 60–69 |
| F | 0–59 |

## RBAC

The governance controller needs patch access to the status subresource:

```yaml
rules:
  - apiGroups: ["agentregistry.dev"]
    resources:
      - mcpservercatalogs/status
      - agentcatalogs/status
      - skillcatalogs/status
      - modelcatalogs/status
    verbs: ["get", "patch", "update"]
  - apiGroups: ["agentregistry.dev"]
    resources:
      - mcpservercatalogs
      - agentcatalogs
      - skillcatalogs
      - modelcatalogs
    verbs: ["get", "list", "watch"]
```

## UI Display

Once patched, the registry UI renders the grade as a color-coded badge on each catalog card:

| Grade | Color |
|---|---|
| A | Green |
| B | Blue |
| C | Yellow |
| D | Orange |
| F | Red |

Hovering the badge shows the numeric score (e.g. `Governance grade: 87/100`).
