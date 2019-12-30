const uploadButton = document.querySelector("#upload-button");
const uploadFile = document.querySelector("#upload-file");

uploadButton.addEventListener("click", () => {
    let data = new FormData()
    for (const file of uploadFile.files) {
        data.append("file", file, file.name)
    }

    fetch(`${API_ENDPOINT}/upload`, {
        method: "PUT",
        body: data,
    })
        .then((data) => data.json())
        .then((parsedJson) => {
            modalAlert("Success", `<p>Your image has been uploaded</p>
            <div class="container">
                <p>Copy the URL below or <a href="${window.location.href}i/${parsedJson.id}">open it</a></p>
                <input readonly value="${window.location.href}i/${parsedJson.id}" style="width:100%;" />
            </div>`);
            uploadFile.value = "";
            loadRecent();
        })
        .catch((err) => modalAlert("Error", err));
});