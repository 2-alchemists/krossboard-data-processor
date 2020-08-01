export KROSSBOARD_LOG_LEVEL="info"
export KROSSBOARD_AZURE_METADATA_SERVICE="http://127.0.0.1:8000"

if [ -z "$AZURE_TENANT_ID" ]; then
    export AZURE_TENANT_ID="{test_6414b971-1319-4421-8952-f98bf160b7f8}"
fi
if [ -z "$AZURE_CLIENT_ID" ]; then
    export AZURE_CLIENT_ID="test_3bd7926c-d7f0-42b4-9844-bc2c4954073d"
fi
if [ -z "$AZURE_CLIENT_SECRET" ]; then
    export AZURE_CLIENT_SECRET="test_]Mo=3lALtZRrqe8f-5rzcu1NQES=9u3T"
fi

make run
