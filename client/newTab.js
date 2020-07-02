(async function pullDetails() {
    const url = "https://reddit-chrome-wallpapers.herokuapp.com/details";
    const response = await fetch(url);
    const data = await response.json();
    chrome.storage.sync.set(data);
}())

chrome.storage.sync.get(null, (items) => {
    console.log(items)
    if (items.Author != null && items.Permalink != null && items.ImageURL != null) {
        const submittedBox = document.getElementById("post_link")
        submittedBox.setAttribute("href", "https://reddit.com" + items.Permalink);
        submittedBox.style.visibility = "visible";
        document.getElementById("author").innerText = "Submitted by u/" + items.Author;
        document.getElementById("body").style.backgroundImage = "url(" + items.ImageURL + ")";
    }
})