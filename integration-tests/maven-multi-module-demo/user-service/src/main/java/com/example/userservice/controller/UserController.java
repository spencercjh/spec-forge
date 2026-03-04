package com.example.userservice.controller;

import com.example.userservice.dto.ApiResponse;
import com.example.userservice.dto.User;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * User management REST API for multi-module project.
 */
@RestController
@RequestMapping("/api/v1/users")
public class UserController {

    private final Map<Long, User> userStore = new HashMap<>();

    public UserController() {
        userStore.put(1L, new User(1L, "alice", "alice@example.com", "Alice Smith", 28));
        userStore.put(2L, new User(2L, "bob", "bob@example.com", "Bob Jones", 32));
        userStore.put(3L, new User(3L, "charlie", "charlie@example.com", "Charlie Brown", 24));
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
     * List all users.
     */
    @GetMapping
    public ApiResponse<List<User>> listUsers(
            @RequestParam(required = false) String username) {

        List<User> allUsers = new ArrayList<>(userStore.values());

        if (username != null && !username.isEmpty()) {
            allUsers.removeIf(u -> !u.getUsername().contains(username));
        }

        return ApiResponse.success(allUsers);
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
     * Delete a user.
     */
    @DeleteMapping("/{id}")
    public ApiResponse<Void> deleteUser(@PathVariable Long id) {
        User removed = userStore.remove(id);
        if (removed == null) {
            return ApiResponse.error(404, "User not found");
        }
        return ApiResponse.success(null);
    }
}
