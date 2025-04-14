. ./env.sh
curl "$API_ENDPOINT/generate?index=0&prompt=A%20Mountain&steps=20&scale=7&sampler_index=DPM%2B%2B%202M%20Karras&width=512&height=512" \
     | jq -r '.image' | base64 -d > output.png