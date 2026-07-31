package shrikeops.security

# METADATA
# title: Deny privileged containers
# description: Containers must not run in privileged mode
# severity: critical
deny[msg] {
    input.kind == "Pod"
    container := input.spec.containers[_]
    container.securityContext.privileged == true
    msg := sprintf("Container '%s' is running privileged — this grants full host access", [container.name])
}

# METADATA
# title: Require non-root user
# description: Containers must not run as root (UID 0)
# severity: high
deny[msg] {
    input.kind == "Pod"
    container := input.spec.containers[_]
    container.securityContext.runAsUser == 0
    msg := sprintf("Container '%s' runs as root (UID 0) — set runAsUser to a non-zero value", [container.name])
}

# METADATA
# title: Require read-only root filesystem
# description: Root filesystem should be read-only to prevent tampering
# severity: medium
warn[msg] {
    input.kind == "Pod"
    container := input.spec.containers[_]
    not container.securityContext.readOnlyRootFilesystem
    msg := sprintf("Container '%s' does not have readOnlyRootFilesystem: true", [container.name])
}

# METADATA
# title: Disallow latest image tag
# description: Using :latest makes deployments non-reproducible
# severity: medium
warn[msg] {
    input.kind == "Deployment"
    container := input.spec.template.spec.containers[_]
    endswith(container.image, ":latest")
    msg := sprintf("Container '%s' uses :latest tag — pin to a specific digest or version", [container.image])
}

# METADATA
# title: Require resource limits
# description: All containers must define CPU and memory limits
# severity: high
deny[msg] {
    input.kind == "Deployment"
    container := input.spec.template.spec.containers[_]
    not container.resources.limits.memory
    msg := sprintf("Container '%s' has no memory limit — risk of OOM eviction", [container.name])
}

deny[msg] {
    input.kind == "Deployment"
    container := input.spec.template.spec.containers[_]
    not container.resources.limits.cpu
    msg := sprintf("Container '%s' has no CPU limit — risk of noisy neighbour", [container.name])
}
