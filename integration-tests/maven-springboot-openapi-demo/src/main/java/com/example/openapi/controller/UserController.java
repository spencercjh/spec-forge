package com.example.openapi.controller;

import com.example.openapi.dto.ApiResponse;
import com.example.openapi.dto.FileUploadResult;
import com.example.openapi.dto.PageResult;
import com.example.openapi.dto.User;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * User management REST API.
 */
@RestController
@RequestMapping("/api/v1/users")
public class UserController {

    private final Map<Long, User> userStore = new HashMap<>();

    public UserController() {
        userStore.put(1L, new User(1L, "john_doe", "john@example.com", "John Doe", 30));
        userStore.put(2L, new User(2L, "jane_smith", "jane@example.com", "Jane Smith", 25));
        userStore.put(3L, new User(3L, "bob_wilson", "bob@example.com", "Bob Wilson", 35));
    }

    /**
     * Get user by ID.
     */
    @GetMapping("/{id}")
    public ApiResponse<User> getUserById(@PathVariable Long id) {
        User user = userStore.get(id);
        if (user == null) {
            return ApiResponse.error(404, "User not found");
        }
        return ApiResponse.success(user);
    }

    /**
     * List users with pagination.
     */
    @GetMapping
    public ApiResponse<PageResult<User>> listUsers(
            @RequestParam(defaultValue = "0") int page,
            @RequestParam(defaultValue = "10") int size,
            @RequestParam(required = false) String username) {

        List<User> allUsers = new ArrayList<>(userStore.values());

        if (username != null && !username.isEmpty()) {
            allUsers.removeIf(u -> !u.getUsername().contains(username));
        }

        int start = page * size;
        int end = Math.min(start + size, allUsers.size());
        List<User> pageContent = allUsers.subList(start, end);

        PageResult<User> result = new PageResult<>();
        result.setContent(pageContent);
        result.setPageNumber(page);
        result.setPageSize(size);
        result.setTotal(allUsers.size());
        result.setTotalPages((int) Math.ceil((double) allUsers.size() / size));

        return ApiResponse.success(result);
    }

    /**
     * Create a new user.
     */
    @PostMapping
    public ApiResponse<User> createUser(@RequestBody User user) {
        Long newId = (long) (userStore.size() + 1);
        user.setId(newId);
        userStore.put(newId, user);
        return ApiResponse.success(user);
    }

    /**
     * Upload a file for a user.
     */
    @PostMapping(value = "/upload", consumes = MediaType.MULTIPART_FORM_DATA_VALUE)
    public ApiResponse<FileUploadResult> uploadFile(
            @RequestParam("file") MultipartFile file,
            @RequestParam(value = "userId", required = false) Long userId) {

        FileUploadResult result = new FileUploadResult();
        result.setFilename(file.getOriginalFilename());
        result.setSize(file.getSize());
        result.setContentType(file.getContentType());
        result.setMessage("File uploaded successfully for user: " + (userId != null ? userId : "anonymous"));

        return ApiResponse.success(result);
    }

    /**
     * Update user profile using form data.
     */
    @PostMapping(value = "/{id}/profile", consumes = MediaType.APPLICATION_FORM_URLENCODED_VALUE)
    public ApiResponse<User> updateProfile(
            @PathVariable Long id,
            @RequestParam(required = false) String fullName,
            @RequestParam(required = false) String email,
            @RequestParam(required = false) Integer age) {

        User user = userStore.get(id);
        if (user == null) {
            return ApiResponse.error(404, "User not found with id: " + id);
        }

        if (fullName != null) {
            user.setFullName(fullName);
        }
        if (email != null) {
            user.setEmail(email);
        }
        if (age != null) {
            user.setAge(age);
        }

        return ApiResponse.success(user);
    }
}
