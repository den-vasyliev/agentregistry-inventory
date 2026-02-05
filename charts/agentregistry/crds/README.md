# Helm CRDs

These are the Agent Registry CRDs installed automatically by Helm before other
chart resources.  They are **not** edited here — they are the source-of-truth
output of `controller-gen` and live in `config/crd/`.

To keep them in sync after running `make generate`:

```bash
cp config/crd/agentregistry.dev_*.yaml charts/agentregistry/crds/
```
