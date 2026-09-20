export NEW_API_KEY=sk-wJrAcW1MTlBgPHJkS3E1zomyFmqtefj6hVsDQflAtmpgvKn7
export MODEL_NAME=Qwen3.8-27B

curl http://10.10.30.8:3002/v1/chat/completions   -H "Authorization: Bearer $NEW_API_KEY"   -H "Content-Type: application/json"   -d '{
       "model": "$MODEL_NAME",
       "messages": [
         {
           "role": "user",
           "content": "Explain quantum entanglement in one paragraph."
         }
       ],
       "temperature": 0.7
     }'