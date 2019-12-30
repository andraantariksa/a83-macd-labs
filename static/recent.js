function loadRecent() {
    document.querySelector("#recent-image").innerHTML = "Loading...";
    fetch(`${API_ENDPOINT}/recent`)
        .then((data) => data.json())
        .then((parsedJson) => {
            let acc = "";
            parsedJson.ids.forEach((val, _) => {
                acc += `<div class="col-md-4"><a href="/i/${val}"><img width="100%" src="${parsedJson.storageURL}/${val}" /></a></div>`;
            });
            document.querySelector("#recent-image").innerHTML = acc;
        })
        .catch((err) => {
            modalAlert("Error", err);
            document.querySelector("#recent-image").innerHTML = `Error happened: ${err}`;
        });
}
loadRecent();