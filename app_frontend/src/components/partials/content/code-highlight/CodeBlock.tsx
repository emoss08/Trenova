/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

/* eslint-disable jsx-a11y/anchor-is-valid */
// @ts-nocheck
import React, {useState, useEffect, useRef} from 'react'
import {CopyToClipboard} from 'react-copy-to-clipboard'
import Highlight, {defaultProps, Language} from 'prism-react-renderer'
import theme from 'prism-react-renderer/themes/shadesOfPurple'
import {OverlayTrigger, Tooltip} from 'react-bootstrap'

type Props = {
  code: string
  language: Language
}

const CodeBlock: React.FC<Props> = ({code, language}) => {
  const codeRef = useRef<HTMLDivElement | null>(null)
  const [copied, setCopied] = useState(false)
  useEffect(() => {
    if (!copied) {
      return
    }

    setTimeout(() => {
      setCopied(false)
    }, 1500)
  }, [copied])

  useEffect(() => {
    if (!codeRef.current) {
      return
    }

    const prismCodeElement = codeRef.current.querySelector('.prism-code ') as HTMLDivElement
    if (prismCodeElement) {
      prismCodeElement.style.background = 'none'
      prismCodeElement.style.fontSize = '13px'
    }
  }, [])

  return (
    <div className='py-5'>
      <div className='highlight'>
        <OverlayTrigger
          key='copy-to-clipboard'
          placement='top'
          overlay={<Tooltip id='tooltip-copy-to-clipboard'>Copy Code</Tooltip>}
        >
          <CopyToClipboard text={code} onCopy={() => setCopied(true)}>
            <a className='highlight-copy btn'>{copied ? 'copied' : 'copy'}</a>
          </CopyToClipboard>
        </OverlayTrigger>

        <div className='highlight-code' ref={codeRef}>
          <Highlight {...defaultProps} theme={theme} code={code} language={language}>
            {({className, style, tokens, getLineProps, getTokenProps}) => {
              return (
                <pre className={className} style={{maxHeight: '300px', ...style}}>
                  {tokens.map((line, i) => (
                    // eslint-disable-next-line react/jsx-key
                    <div {...getLineProps({line, key: i})}>
                      {line.map((token, key) => (
                        // eslint-disable-next-line react/jsx-key
                        <span {...getTokenProps({token, key})} />
                      ))}
                    </div>
                  ))}
                </pre>
              )
            }}
          </Highlight>
        </div>
      </div>
    </div>
  )
}

export {CodeBlock}
