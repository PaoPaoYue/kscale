FROM rayproject/ray:2.9.0-gpu

RUN pip install torch==2.6.0 diffusers==0.33.1 transformers==4.51.2 xformers==0.0.29.post3 accelerate==1.6.0

# Set the working dir for the container to /serve_app
WORKDIR /serve_app

# Copies the local `core` directory to the container's /serve_app directory
COPY core /serve_app/core