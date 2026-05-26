import time
import requests
import json
import argparse

def test_ttft(base_url, api_key, model, prompt_file=None, prompt_text="Hello"):
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}"
    }
    
    content = prompt_text
    if prompt_file:
        try:
            with open(prompt_file, 'r', encoding='utf-8') as f:
                content = f.read()
            print(f"Loaded prompt from {prompt_file} ({len(content)} chars)")
        except Exception as e:
            print(f"Failed to read file: {e}")
            return
            
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": content}],
        "stream": True,
        "max_tokens": 50
    }
    
    # 确保 base_url 末尾没有 /
    if base_url.endswith('/'):
        base_url = base_url[:-1]
    
    endpoint = f"{base_url}/v1/chat/completions"
    print(f"Testing TTFT (首字时长) for model [{model}] at {endpoint} ...")
    start_time = time.time()
    
    try:
        # 发送流式请求
        with requests.post(endpoint, headers=headers, json=payload, stream=True) as response:
            response.raise_for_status()
            
            # 读取流式响应内容
            for line in response.iter_lines():
                if line:
                    line_text = line.decode('utf-8')
                    if line_text.startswith("data: "):
                        data_str = line_text[6:]
                        if data_str == "[DONE]":
                            break
                        try:
                            data = json.loads(data_str)
                            if "choices" in data and len(data["choices"]) > 0:
                                delta = data["choices"][0].get("delta", {})
                                if "content" in delta and delta["content"]:
                                    first_token_time = time.time()
                                    ttft = first_token_time - start_time
                                    print(f"✅ 首字接收内容: '{delta['content']}'")
                                    print(f"⏱️ 首字时长 (TTFT): {ttft:.4f} 秒")
                                    return
                        except json.JSONDecodeError:
                            pass
        
        print("❌ 未能成功读取到首字 stream 输出")
        
    except requests.exceptions.RequestException as e:
        print(f"❌ 请求失败: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Test First Token Latency (TTFT) 首字时长测试脚本")
    parser.add_argument("--url", default="http://localhost:3000", help="API 基础 URL，例如 http://localhost:3000")
    parser.add_argument("--key", required=True, help="API Key (Authorization)")
    parser.add_argument("--model", default="gpt-3.5-turbo", help="请求的模型名称")
    parser.add_argument("--file", default=None, help="读取 prompt 的文件路径，例如 loadtest_200k_tokens.txt")
    parser.add_argument("--prompt", default="你好，测试首字时长", help="如果未提供文件，使用的默认 Prompt")
    
    args = parser.parse_args()
    test_ttft(args.url, args.key, args.model, args.file, args.prompt)
