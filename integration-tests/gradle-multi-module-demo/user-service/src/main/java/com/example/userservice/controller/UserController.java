package com.example.userservice.controller;

import com.example.shared.ApiResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/users")
@Tag(name = "User", description = "User management APIs")
public class UserController {

    @GetMapping
    @Operation(summary = "Get all users", description = "Returns list of all users")
    public ApiResponse<List<String>> getAllUsers() {
        return ApiResponse.success(List.of("user1", "user2", "user3"));
    }

    @GetMapping("/{id}")
    @Operation(summary = "Get user by ID", description = "Returns a single user by ID")
    public ApiResponse<String> getUserById(@PathVariable String id) {
        return ApiResponse.success("user-" + id);
    }
}
