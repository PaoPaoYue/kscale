FROM rayproject/ray:2.9.0

RUN pip install torch diffusers transformers xformers accelerate

# Set the working dir for the container to /serve_app
WORKDIR /serve_app

# Copies the local directory to the container's /serve_app directory
COPY . /serve_app/