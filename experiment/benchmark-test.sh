source env.sh
curl -X POST -F "file=@$BENCHMARK_FILE" $GENERATOR_ENDPOINT/submit-job