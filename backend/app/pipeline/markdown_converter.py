import os
from markitdown import MarkItDown

def convert_to_markdown_local(file_path: str) -> str:
    """
    Converts a document (docx, xlsx, doc, csv, pptx, html, etc.) to markdown using local Microsoft MarkItDown.
    """
    _, ext = os.path.splitext(file_path.lower())
    if ext == '.pdf':
        raise ValueError("PDF files should be sent directly to the vision/document extraction model.")
        
    try:
        md = MarkItDown()
        result = md.convert(file_path)
        return result.text_content
    except Exception as e:
        # Fallback to reading file directly if it's a text/markdown file
        if ext in ['.txt', '.md', '.json', '.csv']:
            try:
                with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                    return f.read()
            except Exception:
                pass
        raise RuntimeError(f"MarkItDown conversion failed: {str(e)}")
