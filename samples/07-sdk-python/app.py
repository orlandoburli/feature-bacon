import os

from flask import Flask, request, jsonify
from feature_bacon import BaconClient, EvaluationContext

app = Flask(__name__)

bacon_url = os.environ.get("BACON_URL", "http://localhost:8080")
api_key = os.environ.get("BACON_API_KEY", "")
environment = os.environ.get("ENVIRONMENT", "production")

client = BaconClient(bacon_url, api_key=api_key)


def user_context() -> EvaluationContext:
    user_id = request.args.get("user", "anonymous")
    plan = request.args.get("plan", "free")
    return EvaluationContext(
        subject_id=user_id,
        environment=environment,
        attributes={"plan": plan, "source": "web"},
    )


@app.get("/")
def home():
    ctx = user_context()
    results = client.evaluate_batch(
        ["dark_mode", "new_pricing", "beta_features", "checkout_redesign", "maintenance_mode"],
        ctx,
    )
    features = {
        r.flag_key: {"enabled": r.enabled, "variant": r.variant, "reason": r.reason}
        for r in results
    }
    return jsonify(service="catalog-api", user=ctx.subject_id, features=features)


@app.get("/products")
def products():
    ctx = user_context()
    new_pricing = client.is_enabled("new_pricing", ctx)
    variant = client.get_variant("checkout_redesign", ctx)

    discount = 0.9 if new_pricing else 1.0
    items = [
        {"id": 1, "name": "Widget Pro", "price": round(29.99 * discount, 2)},
        {"id": 2, "name": "Widget Basic", "price": round(9.99 * discount, 2)},
        {"id": 3, "name": "Widget Enterprise", "price": round(99.99 * discount, 2)},
    ]
    return jsonify(products=items, checkoutVariant=variant, newPricingActive=new_pricing)


@app.get("/health")
def health():
    healthy = client.healthy()
    status = "ok" if healthy else "degraded"
    code = 200 if healthy else 503
    return jsonify(status=status, baconHealthy=healthy), code


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "3000"))
    host = os.environ.get("HOST", "0.0.0.0")  # NOSONAR - runs inside Docker container
    app.run(host=host, port=port)
