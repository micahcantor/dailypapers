(async function pullDetails() {
    const url = "https://reddit-chrome-wallpapers.herokuapp.com/details";
    const response = await fetch(url);
    const data = await response.json();
    chrome.storage.sync.set(data);
}())

chrome.storage.sync.get(null, (items) => {
    if (items.Author != null && items.Permalink != null) {
        document.getElementById("post_link").setAttribute("href", "https://reddit.com" + items.Permalink);
        document.getElementById("author").innerText = "Submitted by u/" + items.Author;
    }
})