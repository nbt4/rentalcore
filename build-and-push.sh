#!/bin/bash

# RentalCore Docker Build and Push Script
# This script builds the Docker image and pushes it to Docker Hub

set -e

# Configuration
DOCKER_USERNAME="nbt4"
IMAGE_NAME="rentalcore"
LATEST_TAG="latest"

# Auto-determine next version number
get_next_version() {
    # Get the latest 1.x version from Docker Hub or local images
    LATEST_VERSION=$(docker images "${DOCKER_USERNAME}/${IMAGE_NAME}" | grep -E "1\.[0-9]+" | head -1 | awk '{print $2}' | sed 's/1\.//')
    
    if [ -z "$LATEST_VERSION" ]; then
        # No 1.x versions found, start with 1.1
        echo "1.1"
    else
        # Increment the minor version
        NEXT_MINOR=$((LATEST_VERSION + 1))
        if [ "$NEXT_MINOR" -eq 10 ]; then
            echo "2.0"  # After 1.9 comes 2.0
        else
            echo "1.$NEXT_MINOR"
        fi
    fi
}

VERSION="${1:-$(get_next_version)}"  # Use argument or auto-generated version

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE} RentalCore Docker Build & Push${NC}"
    echo -e "${BLUE}================================================${NC}"
}

# Check if Docker username is set
check_username() {
    if [ -z "$DOCKER_USERNAME" ]; then
        print_error "Docker username not set!"
        echo "Please edit this script and set your Docker Hub username:"
        echo "DOCKER_USERNAME=\"your-username\""
        exit 1
    fi
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running or accessible"
        exit 1
    fi
    print_success "Docker is running"
}

# Check if logged in to Docker Hub
check_login() {
    if ! docker info | grep -q Username; then
        print_warning "Not logged in to Docker Hub"
        print_info "Please login to Docker Hub:"
        docker login
    else
        print_success "Already logged in to Docker Hub"
    fi
}

# Build the Docker image
build_image() {
    print_info "Building Docker image..."
    
    # Build with latest tag
    print_info "Building ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}..."
    docker build -t "${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}" .
    
    # Build with version tag
    print_info "Building ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}..."
    docker build -t "${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}" .
    
    print_success "Docker images built successfully"
}

# Push images to Docker Hub
push_images() {
    print_info "Pushing images to Docker Hub..."
    
    # Push latest
    print_info "Pushing ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}..."
    docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}"
    
    # Push version
    print_info "Pushing ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}..."
    docker push "${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}"
    
    print_success "Images pushed successfully"
}

# Clean up local images (optional)
cleanup() {
    read -p "Do you want to remove local images to save space? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Removing local images..."
        docker rmi "${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}" || true
        docker rmi "${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}" || true
        print_success "Local images removed"
    fi
}

# Display final information
show_info() {
    echo ""
    print_header
    echo -e "${GREEN}✅ Build and push completed successfully!${NC}"
    echo ""
    print_info "Your images are now available on Docker Hub:"
    echo "  • ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}"
    echo "  • ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}"
    echo ""
    print_info "To deploy on another system:"
    echo "  1. Copy docker-compose.prod.yml and .env.template"
    echo "  2. Update the image name in docker-compose.prod.yml:"
    echo "     image: ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}"
    echo "  3. Configure .env file with your settings"
    echo "  4. Run: docker-compose -f docker-compose.prod.yml up -d"
    echo ""
    print_info "Docker Hub URL: https://hub.docker.com/r/${DOCKER_USERNAME}/${IMAGE_NAME}"
    echo ""
}

# Handle script arguments
case "${1:-}" in
    --set-username)
        if [ -n "$2" ]; then
            sed -i "s/DOCKER_USERNAME=\"\"/DOCKER_USERNAME=\"$2\"/" "$0"
            print_success "Docker username set to: $2"
            exit 0
        else
            print_error "Please provide a username: $0 --set-username your-username"
            exit 1
        fi
        ;;
    --help|-h)
        echo "RentalCore Docker Build & Push Script"
        echo ""
        echo "Usage:"
        echo "  $0 [VERSION]          Build and push images with version (default: timestamp)"
        echo "  $0 --set-username USER Set Docker Hub username"
        echo "  $0 --help             Show this help"
        echo ""
        echo "Examples:"
        echo "  $0 1.11               Build version 1.11"
        echo "  $0                    Build with timestamp version"
        echo ""
        exit 0
        ;;
esac

# Main execution
main() {
    print_header
    
    check_username
    check_docker
    check_login
    
    print_info "Building version: ${VERSION}"
    print_info "Target repository: ${DOCKER_USERNAME}/${IMAGE_NAME}"
    
    build_image
    push_images
    cleanup
    show_info
}

# Run main function
main "$@"