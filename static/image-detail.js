const imageInfo = document.querySelector("#image-info");

function loadImageDetail(id) {
    fetch(`${API_ENDPOINT}/detail/${id}`)
        .then((data) => data.json())
        .then((parsedJson) => {
            imageInfo.innerHTML = "";
            imageInfo.innerHTML += "<h3>Captions</h3><ul>";
            parsedJson.captions.forEach((val, idx) => {
                imageInfo.innerHTML += `<li>${val.text} – ${Math.round(val.confidence * 100.0)}%</li>`
            });
            imageInfo.innerHTML += "</ul><h3>Tags</h3>";
            parsedJson.tags.forEach((val, idx) => {
                imageInfo.innerHTML += `<li>${val}</li>`
            });
            imageInfo.innerHTML += "</ul>"
        })
        .catch((err) => modalAlert("Error", err));
}
loadImageDetail(imageID);