let videos = [];

async function loadVideos() {
    try {
        document.getElementById('status').textContent = 'Loading videos...';
        const response = await fetch('/api/videos');
        videos = await response.json();
        
        if (videos.length === 0) {
            document.getElementById('status').textContent = 'No videos uploaded yet';
        } else {
            document.getElementById('status').textContent = `${videos.length} video(s) available`;
        }
        
        renderVideos();
    } catch (error) {
        console.error('Error loading videos:', error);
        document.getElementById('status').textContent = 'Error loading videos';
    }
}

function renderVideos() {
    const grid = document.getElementById('video-grid');
    grid.innerHTML = '';

    videos.forEach(video => {
        const item = document.createElement('div');
        item.className = 'video-item';
        
        const name = document.createElement('div');
        name.className = 'video-name';
        name.textContent = video;

        const previewContainer = document.createElement('div');
        previewContainer.className = 'preview-container';
        
        const placeholder = document.createElement('div');
        placeholder.className = 'preview-placeholder';
        placeholder.textContent = 'Hover to preview';
        previewContainer.appendChild(placeholder);

        item.appendChild(previewContainer);
        item.appendChild(name);

        // Hover preview
        item.addEventListener('mouseenter', () => {
            previewContainer.innerHTML = '';
            const previewVideo = document.createElement('video');
            previewVideo.src = `/preview/preview_${video}`;
            previewVideo.autoplay = true;
            previewVideo.muted = true;
            previewVideo.loop = true;
            previewContainer.appendChild(previewVideo);
        });

        item.addEventListener('mouseleave', () => {
            previewContainer.innerHTML = '';
            const placeholder = document.createElement('div');
            placeholder.className = 'preview-placeholder';
            placeholder.textContent = 'Hover to preview';
            previewContainer.appendChild(placeholder);
        });

        item.addEventListener('click', () => {
            openModal(video);
        });

        grid.appendChild(item);
    });
}

function openModal(videoName) {
    const modal = document.getElementById('video-modal');
    const video = document.getElementById('modal-video');
    video.src = `/video/${videoName}`;
    modal.classList.add('active');
    video.play();
}

function closeModal() {
    const modal = document.getElementById('video-modal');
    const video = document.getElementById('modal-video');
    video.pause();
    video.src = '';
    modal.classList.remove('active');
}

document.getElementById('video-modal').addEventListener('click', (e) => {
    if (e.target.id === 'video-modal') {
        closeModal();
    }
});

loadVideos();
setInterval(loadVideos, 5000);