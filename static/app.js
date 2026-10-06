const video = document.getElementById('webcam');
const canvas = document.getElementById('captureCanvas');
const btnStart = document.getElementById('btnStart');
const btnStop = document.getElementById('btnStop');
const intervalSelect = document.getElementById('intervalSelect');
const statusBadge = document.getElementById('statusBadge');
const frameCounter = document.getElementById('frameCounter');
const timerDisplay = document.getElementById('timerDisplay');
const cameraContainer = document.querySelector('.camera-container');

// Flash element
const flashEl = document.createElement('div');
flashEl.className = 'flash';
cameraContainer.appendChild(flashEl);

let captureInterval = null;
let timerInterval = null;
let frames = 0;
let startTime = 0;

// Initialize Webcam
async function initWebcam() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({
            video: {
                width: { ideal: 1920 },
                height: { ideal: 1080 },
                facingMode: "user"
            }
        });
        video.srcObject = stream;
    } catch (err) {
        console.error("Error accessing webcam:", err);
        alert("Could not access the webcam. Please ensure you have granted permission.");
    }
}

// Format time utility
function formatTime(ms) {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
    const s = (totalSeconds % 60).toString().padStart(2, '0');
    return `${m}:${s}`;
}

// Trigger visual flash
function triggerFlash() {
    flashEl.classList.remove('flash-anim');
    void flashEl.offsetWidth; // trigger reflow
    flashEl.classList.add('flash-anim');
}

// Capture frame and send to server
async function captureFrame() {
    if (!video.videoWidth) return;

    // Set canvas dimensions to match video
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    
    const ctx = canvas.getContext('2d');
    
    // Draw current frame to canvas
    // We draw it mirrored because we display it mirrored, but usually you want actual orientation
    // For now we just draw it as is (which is unmirrored data-wise, standard for cameras)
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    
    // Get base64 jpeg
    const dataUrl = canvas.toDataURL('image/jpeg', 0.9);
    
    try {
        triggerFlash();
        frames++;
        frameCounter.textContent = `Frames: ${frames}`;

        await fetch('/upload', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ image: dataUrl })
        });
    } catch (err) {
        console.error("Failed to upload frame", err);
    }
}

// Update running timer
function updateTimer() {
    const elapsed = Date.now() - startTime;
    timerDisplay.textContent = formatTime(elapsed);
}

// Start Timelapse
btnStart.addEventListener('click', () => {
    const intervalSeconds = parseInt(intervalSelect.value);
    
    btnStart.disabled = true;
    btnStop.disabled = false;
    intervalSelect.disabled = true;
    
    statusBadge.textContent = "Recording";
    statusBadge.classList.add('active');
    
    frames = 0;
    frameCounter.textContent = `Frames: 0`;
    startTime = Date.now();
    updateTimer();
    
    // Capture first frame immediately
    captureFrame();
    
    // Set intervals
    captureInterval = setInterval(captureFrame, intervalSeconds * 1000);
    timerInterval = setInterval(updateTimer, 1000);
});

const btnRender = document.getElementById('btnRender');

// Stop Timelapse
btnStop.addEventListener('click', () => {
    clearInterval(captureInterval);
    clearInterval(timerInterval);
    
    btnStart.disabled = false;
    btnStop.disabled = true;
    intervalSelect.disabled = false;
    
    if (frames > 0) {
        btnRender.style.display = 'inline-flex';
    }
    
    statusBadge.textContent = "Idle";
    statusBadge.classList.remove('active');
});

// Render Video
btnRender.addEventListener('click', async () => {
    const originalText = btnRender.textContent;
    btnRender.textContent = "Rendering...";
    btnRender.disabled = true;
    
    try {
        const response = await fetch('/compile', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fps: 10 })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            alert(`Video successfully rendered!\nSaved to: ${result.file}`);
            btnRender.style.display = 'none';
            frames = 0;
            frameCounter.textContent = `Frames: 0`;
            timerDisplay.textContent = "00:00";
        } else {
            alert(`Error: ${result.message || 'Failed to render video'}`);
        }
    } catch (err) {
        console.error("Compile error:", err);
        alert("An error occurred during rendering.");
    } finally {
        btnRender.textContent = originalText;
        btnRender.disabled = false;
    }
});

// Initialize on load
window.addEventListener('DOMContentLoaded', initWebcam);
