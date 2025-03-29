. ./env.sh
curl -o "result/${TEST_NAME}-result.csv" "$GENERATOR_ENDPOINT/download-result?batchname=$TEST_NAME"
