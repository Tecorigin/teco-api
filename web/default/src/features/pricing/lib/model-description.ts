import DOMPurify from 'dompurify'
import { marked } from 'marked'

export function getModelDescriptionPreview(description?: string): string {
  if (!description) return ''

  const template = document.createElement('template')
  template.innerHTML = DOMPurify.sanitize(
    marked.parse(description, { async: false, gfm: true }),
    { FORBID_TAGS: ['script', 'style'] }
  )
  template.content.querySelectorAll('img').forEach((image) => {
    image.replaceWith(image.alt)
  })
  template.content
    .querySelectorAll('p,h1,h2,h3,h4,h5,h6,li,pre,blockquote,tr,br,hr')
    .forEach((element) => element.after(' '))

  return (template.content.textContent ?? '').replaceAll(/\s+/g, ' ').trim()
}
