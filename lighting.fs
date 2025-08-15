#version 330

// Input vertex attributes (from vertex shader)
in vec3 fragPosition;
in vec2 fragTexCoord;
in vec3 fragNormal;
in vec4 fragColor;

// Input uniform values
uniform sampler2D texture0;
uniform vec4 colDiffuse;

// Lighting uniforms
uniform vec3 viewPos;
uniform int lightCount;

// Global lighting
uniform vec3 globalLightColor;
uniform float globalLightIntensity;

// Sun lighting (directional)
uniform vec3 sunDirection;
uniform vec3 sunColor;
uniform float sunIntensity;

struct Light {
    vec3 position;
    vec3 color;
    float intensity;
};

uniform Light lights[8]; // Maximum 8 lights

// Output fragment color
out vec4 finalColor;

// Calculate attenuation for point lights
float calculateAttenuation(float distance) {
    float constant = 1.0;
    float linear = 0.09;
    float quadratic = 0.032;
    
    return 1.0 / (constant + linear * distance + quadratic * (distance * distance));
}

// Calculate diffuse lighting
vec3 calculateDiffuse(vec3 lightDir, vec3 normal, vec3 lightColor, float intensity) {
    float diff = max(dot(normal, lightDir), 0.0);
    return diff * lightColor * intensity;
}

// Calculate specular lighting
vec3 calculateSpecular(vec3 lightDir, vec3 normal, vec3 viewDir, vec3 lightColor, float intensity) {
    vec3 reflectDir = reflect(-lightDir, normal);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), 32.0); // shininess = 32
    return spec * lightColor * intensity * 0.5; // reduce specular intensity
}

// Simple shadow calculation (basic implementation)
float calculateShadow(vec3 fragPos, vec3 lightPos) {
    // This is a simplified shadow - for real shadows you'd need shadow mapping
    // For now, just create some basic occlusion based on distance and angle
    return 1.0; // No shadows for now - can be expanded later
}

void main()
{
    // Get base color from texture and vertex color
    vec4 texelColor = texture(texture0, fragTexCoord);
    vec3 baseColor = texelColor.rgb * fragColor.rgb * colDiffuse.rgb;
    
    // Normalize the fragment normal
    vec3 normal = normalize(fragNormal);
    vec3 viewDir = normalize(viewPos - fragPosition);
    
    // Global ambient lighting
    vec3 ambient = globalLightColor * globalLightIntensity * baseColor * 0.15;
    
    // Initialize lighting accumulation
    vec3 diffuse = vec3(0.0);
    vec3 specular = vec3(0.0);
    
    // Sun lighting (directional light)
    if(sunIntensity > 0.0) {
        vec3 sunDir = normalize(-sunDirection); // Negate for light direction
        
        // Sun diffuse
        float sunDiff = max(dot(normal, sunDir), 0.0);
        diffuse += sunDiff * sunColor * sunIntensity;
        
        // Sun specular
        vec3 sunReflectDir = reflect(-sunDir, normal);
        float sunSpec = pow(max(dot(viewDir, sunReflectDir), 0.0), 64.0);
        specular += sunSpec * sunColor * sunIntensity * 0.3;
    }
    
    // Point lights
    for(int i = 0; i < lightCount && i < 8; i++) {
        vec3 lightPos = lights[i].position;
        vec3 lightColor = lights[i].color;
        float lightIntensity = lights[i].intensity;
        
        // Calculate light direction and distance
        vec3 lightDir = lightPos - fragPosition;
        float distance = length(lightDir);
        lightDir = normalize(lightDir);
        
        // Calculate attenuation
        float attenuation = calculateAttenuation(distance);
        
        // Calculate shadow factor
        float shadow = calculateShadow(fragPosition, lightPos);
        
        // Calculate lighting components
        vec3 lightDiffuse = calculateDiffuse(lightDir, normal, lightColor, lightIntensity);
        vec3 lightSpecular = calculateSpecular(lightDir, normal, viewDir, lightColor, lightIntensity);
        
        // Apply attenuation and shadow
        diffuse += lightDiffuse * attenuation * shadow;
        specular += lightSpecular * attenuation * shadow;
    }
    
    // Combine all lighting components
    vec3 result = ambient + diffuse + specular;
    
    // Apply base color
    result *= baseColor;
    
    // Output final color with original alpha
    finalColor = vec4(result, texelColor.a * fragColor.a * colDiffuse.a);
}