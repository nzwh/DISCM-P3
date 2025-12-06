let videos = [];

async function loadVideos() {
    try {
        const response = await fetch('/api/videos');
        const newVideos = await response.json();
        
        if (newVideos.length === 0) {
            document.getElementById('status').textContent = 'No videos uploaded yet';
        } else {
            document.getElementById('status').textContent = `${newVideos.length} video(s) available`;
        }
        
        const hasChanged = JSON.stringify(videos) !== JSON.stringify(newVideos);
        if (hasChanged) {
            videos = newVideos;
            renderVideos();
        }
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
        
        const previewVideo = document.createElement('video');
        previewVideo.src = `/preview/preview_${video}`;
        previewVideo.muted = true;
        previewVideo.loop = true;
        previewVideo.playsInline = true;
        previewVideo.preload = 'metadata';
        previewVideo.style.display = 'none';
        
        previewContainer.appendChild(placeholder);
        previewContainer.appendChild(previewVideo);

        item.appendChild(previewContainer);
        item.appendChild(name);
        item.addEventListener('mouseenter', () => {
            placeholder.style.display = 'none';
            previewVideo.style.display = 'block';
            previewVideo.currentTime = 0;
            previewVideo.play().catch(err => {
                console.log('Preview play failed:', err);
            });
        });

        item.addEventListener('mouseleave', () => {
            previewVideo.pause();
            previewVideo.style.display = 'none';
            placeholder.style.display = 'block';
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