import os
import docx

def convert_to_markdown_local(file_path: str) -> str:
    """
    Converts a document (docx, txt, md, csv, etc.) to markdown/text using local lightweight libraries.
    """
    _, ext = os.path.splitext(file_path.lower())
    if ext == '.pdf':
        raise ValueError("PDF files should be sent directly to the vision/document extraction model.")
        
    try:
        if ext == '.docx':
            doc = docx.Document(file_path)
            paragraphs = []
            for p in doc.paragraphs:
                if p.text.strip():
                    paragraphs.append(p.text)
            return "\n\n".join(paragraphs)
            
        elif ext in ['.txt', '.md', '.json', '.csv']:
            with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                return f.read()
                
        else:
            # Catch-all fallback for other text-like formats
            with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                return f.read()
    except Exception as e:
        raise RuntimeError(f"Document conversion failed for {ext}: {str(e)}")
