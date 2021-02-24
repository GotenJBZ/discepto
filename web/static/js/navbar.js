document.addEventListener('DOMContentLoaded', () => {
    const $navbarBurgers = Array.prototype.slice.call(document.querySelectorAll('.navbar-burger'), 0);
    if ($navbarBurgers.length > 0) {
        $navbarBurgers.forEach(el => {
            el.addEventListener('click', () => {
                const target = el.dataset.target;
                const $target = document.getElementById(target);
                el.classList.toggle('is-active');
                $target.classList.toggle('is-active');
            });
        });
    }
});





class BulmaModal {
    constructor(selector) {
        this.elem = document.querySelector(selector)
        this.close_data()
    }
    show() {
        this.elem.classList.toggle('is-active')
        this.on_show()
    }
    close() {
        this.elem.classList.toggle('is-active')
        this.on_close()
    }
    close_data() {
        var modalClose = this.elem.querySelectorAll("[data-bulma-modal='close'], .modal-background ")
        var that = this
        modalClose.forEach(function(e) {
            console.log(e)
            e.addEventListener("click", function() {
                that.elem.classList.toggle('is-active')
                var event = new Event('modal:close')
                that.elem.dispatchEvent(event);
            })
        })
    }
    on_show() {
        var event = new Event('modal:show')
        this.elem.dispatchEvent(event);
    }
    on_close() {
        var event = new Event('modal:close')
        this.elem.dispatchEvent(event);
    }
    addEventListener(event, callback) {
        this.elem.addEventListener(event, callback)
    }
}

var btn = document.querySelector("#searchBtn")
var mdl = new BulmaModal("#searchModal")

btn.addEventListener("click", function() {
    mdl.show()
});