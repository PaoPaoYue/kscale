. ./env.sh
curl -X POST "$API_ENDPOINT/sdapi/v1/txt2img" \
     -H "Content-Type: application/json" \
     -d '{"prompt": "a mountain", "sampler_index": "DPM++ 2M", "sceduler": "Automatic", "steps": 20, "cfg_scale": 7.5, "width": 512, "height": 512}' \
     | jq -r '.images[0]' | base64 -d > output.png