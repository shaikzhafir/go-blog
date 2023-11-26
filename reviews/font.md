---
Title: Changing font with tailwindcss
Summary: how to easily use fonts you see online in your sites
Published: 29-10-2023
Slug: font-meister
Tags:
  - coding
---

## when you just wanna use some random ass font

I updated the font to use geist, a font developed by vercel. 

Download the otf zip from this link: [geist download link]("https://github.com/vercel/geist-font/releases "click me i wont bite")<br><br>

You then modify your css file to add a path to the .otf file <br><br>

```css
@font-face {
  font-family: 'geist';
  src: url("./public/fonts/Geist-Regular.otf");
}


```

<br>
Im using tailwindcss cos I suck at CSS, so i have to modify my tailwind config to add the font<br><br>

```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./*/*.html', '*.html'],
  theme: {
    extend: {
      fontFamily: {
        geist: ['geist', 'sans-serif'],
      },
    },
  },

  plugins: [
    require('@tailwindcss/typography'),
  ],
}


```
<br><br>

The final touch, is just adding the font to your html file as easily as <br><br>

```html
  <body>
    <div class="py-20 mx-5 xl:mx-80 bg-slate-500 mb-10 font-geist">
      <a href="/">
        <p class="text-3xl my-2 text-green-500">welcome to szhafir blog</p>
      </a>
        <p class="font-bold text-2xl mt-2">{{.Title}}</h2>
        <p class="mb-10">{{.Published}}</h3>
        {{.Content}}
    </div>
  </body>
</html>

```

<br><br>

This post was entirely made possible by using chatgpt. its really the age of idea guys who have a bit of patience to actually try and build stuff. 