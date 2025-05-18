. ./env.sh
curl -o "result/${TEST_NAME}-metrics.csv" "$GENERATOR_ENDPOINT/download-metrics?batchname=$TEST_NAME"
