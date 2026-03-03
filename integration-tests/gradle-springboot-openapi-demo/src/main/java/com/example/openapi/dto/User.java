package com.example.openapi.dto;

/**
 * User entity representing a user in the system.
 */
public class User {

    private Long id;
    private String username;
    private String email;
    private String fullName;
    private Integer age;

    public User() {
    }

    public User(Long id, String username, String email, String fullName, Integer age) {
        this.id = id;
        this.username = username;
        this.email = email;
        this.fullName = fullName;
        this.age = age;
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public String getEmail() {
        return email;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    public String getFullName() {
        return fullName;
    }

    public void setFullName(String fullName) {
        this.fullName = fullName;
    }

    public Integer getAge() {
        return age;
    }

    public void setAge(Integer age) {
        this.age = age;
    }
}
