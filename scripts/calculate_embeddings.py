import os
import sys
import json
import urllib.request
import numpy as np

# Ensure dependencies are available
try:
    import onnxruntime as ort
    from tokenizers import Tokenizer
except ImportError:
    print(json.dumps({"error": "Missing dependencies: run pip install onnxruntime tokenizers"}))
    sys.exit(1)

CACHE_DIR = os.environ.get("P2B_MODEL_CACHE_DIR", os.path.expanduser("~/.cache/p2b-embeddings"))
MODEL_URL = "https://huggingface.co/Xenova/multilingual-e5-base/resolve/main/onnx/model_quantized.onnx"
TOKENIZER_URL = "https://huggingface.co/Xenova/multilingual-e5-base/resolve/main/tokenizer.json"

MODEL_PATH = os.path.join(CACHE_DIR, "model_quantized.onnx")
TOKENIZER_PATH = os.path.join(CACHE_DIR, "tokenizer.json")

def download_file(url, dest_path):
    os.makedirs(os.path.dirname(dest_path), exist_ok=True)
    temp_dest = dest_path + ".tmp"
    try:
        urllib.request.urlretrieve(url, temp_dest)
        os.replace(temp_dest, dest_path)
    except Exception as e:
        if os.path.exists(temp_dest):
            os.remove(temp_dest)
        raise e

def load_model_and_tokenizer():
    if not os.path.exists(TOKENIZER_PATH):
        download_file(TOKENIZER_URL, TOKENIZER_PATH)
    if not os.path.exists(MODEL_PATH):
        download_file(MODEL_URL, MODEL_PATH)
        
    tokenizer = Tokenizer.from_file(TOKENIZER_PATH)
    
    # Configure ONNX Runtime session for low-resource CPU execution
    sess_options = ort.SessionOptions()
    sess_options.intra_op_num_threads = 2
    sess_options.inter_op_num_threads = 2
    sess_options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
    
    session = ort.InferenceSession(MODEL_PATH, sess_options, providers=['CPUExecutionProvider'])
    return tokenizer, session

def mean_pooling(last_hidden_state, attention_mask):
    input_mask_expanded = np.expand_dims(attention_mask, axis=-1).astype(float)
    sum_embeddings = np.sum(last_hidden_state * input_mask_expanded, axis=1)
    sum_mask = np.clip(input_mask_expanded.sum(axis=1), a_min=1e-9, a_max=None)
    return sum_embeddings / sum_mask

def main():
    # Read text from stdin to prevent command-line length limits
    try:
        input_data = sys.stdin.read().strip()
        if not input_data:
            print(json.dumps({"error": "No input text provided via stdin"}))
            sys.exit(1)
            
        tokenizer, session = load_model_and_tokenizer()
        
        # Tokenize (E5 requires "passage: " prefix for documents)
        encoded = tokenizer.encode(input_data)
        
        input_ids = np.array([encoded.ids], dtype=np.int64)
        attention_mask = np.array([encoded.attention_mask], dtype=np.int64)
        token_type_ids = np.array([encoded.type_ids], dtype=np.int64)
        
        # Build inputs based on what the ONNX model expects
        ort_inputs = {
            "input_ids": input_ids,
            "attention_mask": attention_mask
        }
        
        expected_inputs = [inp.name for inp in session.get_inputs()]
        if "token_type_ids" in expected_inputs:
            ort_inputs["token_type_ids"] = token_type_ids
        
        # Run ONNX inference
        ort_outputs = session.run(None, ort_inputs)
        last_hidden_state = ort_outputs[0]
        
        # Mean pooling
        pooled = mean_pooling(last_hidden_state, attention_mask)[0]
        
        # L2 Normalization
        norm = np.linalg.norm(pooled)
        normalized = (pooled / (norm if norm > 0 else 1e-9)).tolist()
        
        # Print output to stdout
        print(json.dumps(normalized))
        
    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)

if __name__ == "__main__":
    main()
