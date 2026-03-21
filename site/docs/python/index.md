---
sidebar_label: Python SDK
sidebar_position: 3
---

# Python SDK

Documentation for the Cuttlegate Python SDK is coming soon.

The Python SDK will provide synchronous and async clients for evaluating feature flags in Python services.

```python
# Coming soon
from cuttlegate import Client

client = Client(base_url="https://your-instance", api_key=api_key)
result = client.evaluate_flag("my-flag", context={"user_id": "user-123"})
```

In the meantime, see the [API reference](https://github.com/karo/cuttlegate) for the raw HTTP evaluation endpoint.
