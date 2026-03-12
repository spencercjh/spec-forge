package com.example.shared;

public class FileUploadResult {
    private String filename;
    private long size;
    private String contentType;
    private String message;

    public FileUploadResult() {}

    public String getFilename() { return filename; }
    public void setFilename(String filename) { this.filename = filename; }
    public long getSize() { return size; }
    public void setSize(long size) { this.size = size; }
    public String getContentType() { return contentType; }
    public void setContentType(String contentType) { this.contentType = contentType; }
    public String getMessage() { return message; }
    public void setMessage(String message) { this.message = message; }
}
