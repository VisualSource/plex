package devtools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
)

func StartRemoteDebuggingServer(logger *slog.Logger, ctx context.Context, port int) error {
	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.DebugContext(ctx, "starting remote devtools", slog.Int("port", port))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			logger.ErrorContext(ctx, "failed to accept connection", slog.String("error", err.Error()))
			continue
		}

		go handleConnection(logger, ctx, conn)
	}

	return nil
}

func handleConnection(logger *slog.Logger, ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	err := writePacket(writer, map[string]any{
		"from":            "root",
		"applicationType": "browser",
		"traits": map[string]bool{
			"networkMonitor": false,
		},
	})
	if err != nil {
		logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
		return
	}

	logger.DebugContext(ctx, "new connection", slog.String("remote", conn.RemoteAddr().String()))

	for {
		message, err := readPacket(reader)
		if err != nil {
			logger.ErrorContext(ctx, "packet read error", slog.String("error", err.Error()))
			return
		}

		logger.DebugContext(ctx, "read packet", slog.Any("packet", message))

		switch message["to"].(string) {
		case "root":
			switch message["type"].(string) {
			case "connect":
				if err := writePacket(writer, map[string]any{
					"from": "root",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getRoot":
				//https://searchfox.org/firefox-main/source/devtools/shared/specs/root.js
				if err := writePacket(writer, map[string]any{
					"from":            "root",
					"preferenceActor": "plex.conn0.preference",
					"deviceActor":     "plex.conn0.device1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listTabs":
				if err := writePacket(writer, map[string]any{
					"from":     "root",
					"selected": 0,
					"tabs": []map[string]any{
						{
							"actor":             "plex.conn0.tabDescriptor1",
							"browserId":         1,
							"browsingContextID": nil,
							"outerWindowID":     1,
							"selected":          true,
							"traits": map[string]bool{
								"watcher":                  true,
								"supportsReloadDescriptor": false,
								"supportsNavigation":       false,
							},
							"title": "Plex",
							"url":   "about:blank",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listAddons":
				if err := writePacket(writer, map[string]any{
					"from":   "root",
					"addons": []map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getTab":
				// borwserId is also herer
				if err := writePacket(writer, map[string]any{
					"from": "root",
					"tab": map[string]any{
						"actor":             "plex.conn0.tabDescriptor1",
						"browserId":         message["browserId"].(float64),
						"browsingContextID": nil,
						"isZombieTab":       false,
						"outerWindowID":     1,
						"selected":          true,
						"title":             "Plex",
						"traits": map[string]bool{
							"watcher":                  true, // needed for firefox 102+
							"supportsReloadDescriptor": false,
							"supportsNavigation":       false,
						},
						"url": "about:blank",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
				logger.DebugContext(ctx, "sent packet")
			}
		case "plex.conn0.device1":
			switch message["type"].(string) {
			case "getDescription":
				if err := writePacket(writer, map[string]any{
					"from": "plex.conn0.device1",
					"value": map[string]any{
						"apptype":   "browser",
						"name":      "Plex",
						"vender":    "Plex",
						"brandName": "Plex",
						"version":   "0.1.0",
						"channel":   "release",
						"os":        "Linux",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				}
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
				logger.DebugContext(ctx, "sent packet")
			}
		case "plex.conn0.preference":
			switch message["type"].(string) {
			case "getBoolPref":
				if err := writePacket(writer, map[string]any{
					"from":  "plex.conn0.preference",
					"value": false,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				}
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
				logger.DebugContext(ctx, "sent packet")
			}
		case "plex.conn0.tabDescriptor1":
			switch message["type"].(string) {
			case "getWatcher":
				err := writePacket(writer, map[string]any{
					"from":  message["to"],
					"actor": "plex.conn0.watcher1",
					"traits": map[string]any{
						"frame":          true,
						"process":        false,
						"worker":         false,
						"service_worker": false,
						"shared_worker":  false,
						"content_script": false,
						"resources": map[string]bool{
							"console-message": false,
							"document-event":  false,
							"error-message":   false,
							"source":          true,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.watcher1":
			switch message["type"].(string) {
			case "watchTargets":
				targetType := message["targetType"].(string)

				if targetType == "frame" {
					err := writePacket(writer, map[string]any{
						"from": message["to"],
						"type": "target-available-form",
						"target": map[string]any{
							"actor":                       "plex.conn0.windowGlobalTarget1",
							"targetType":                  "frame",
							"browsingContextID":           1,
							"processID":                   1,
							"followWindowGlobalLifeCycle": true,
							"innerWindowId":               1,
							"parentInnerWindowId":         0,
							"topInnerWindowId":            1,
							"isTopLevelTarget":            true,
							"ignoreSubFrames":             false,
							"isPopup":                     false,
							"isPrivate":                   false,
							"title":                       "Plex",
							"url":                         "about:blank",
							"outerWindowID":               1,
							"isFallbackExtensionDocument": false,
							"addonId":                     nil,

							"traits": map[string]bool{
								"isBrowsingContext":          true,
								"supportsTopLevelTargetFlag": true,
								"frames":                     true,
								"logInPage":                  true,
								"watchpoints":                true,
								"navigation":                 true,
							},

							// Sub-actor IDs — only advertise what you implement.
							// Each one becomes a new "to" target the client will start calling.
							//"consoleActor":   "plex.conn0.console1",
							//"threadActor":    "plex.conn0.thread1",
							"inspectorActor": "plex.conn0.inspector1",
						},
					})
					if err != nil {
						logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
						return
					}
				}

				if err := writePacket(writer, map[string]any{
					"from": "plex.conn0.watcher1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getThreadConfigurationActor":

				if err := writePacket(writer, map[string]any{
					"from": "plex.conn0.watcher1",
					"configuration": map[string]any{
						"actor": "plex.conn0.threadConfiguration1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getTargetConfigurationActor":
				if err := writePacket(writer, map[string]any{
					"from": "plex.conn0.watcher1",
					"configuration": map[string]any{
						"actor":         "plex.conn0.targetConfiguration1",
						"configuration": map[string]any{},
						"traits": map[string]any{
							"supportedOptions": map[string]bool{},
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getParentBrowsingContextID":
				if err := writePacket(writer, map[string]any{
					"from":              "plex.conn0.watcher1",
					"browsingContextID": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.targetConfiguration1", "plex.conn0.threadConfiguration1":
			switch message["type"] {
			case "updateConfiguration":
				err := writePacket(writer, map[string]any{
					"from": message["to"],
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}

		case "plex.conn0.inspector1":
			switch message["type"].(string) {
			case "getWalker":
				err := writePacket(writer, map[string]any{
					"from": message["to"],
					"walker": map[string]any{
						"actor":             "plex.conn0.walker1",
						"rfpCSSColorScheme": false,
						"traits":            map[string]any{},
						"root": map[string]any{
							"actor":                   "plex.conn0.node1",
							"baseURI":                 "about:blank",
							"parent":                  nil,
							"nodeType":                9, // DOCUMENT_NODE
							"namespaceURI":            "http://www.w3.org/1999/xhtml",
							"nodeName":                "#document",
							"nodeValue":               nil,
							"displayName":             "#document",
							"numChildren":             1, // 1 = the <html> element
							"displayType":             nil,
							"isScrollable":            false,
							"isTopLevelDocument":      true,
							"causesOverflow":          false,
							"containerType":           nil,
							"anchorName":              nil,
							"attrs":                   []any{},
							"customElementLocation":   nil,
							"isPseudoElement":         false,
							"isNativeAnonymous":       false,
							"isShadowRoot":            false,
							"shadowRootMode":          nil,
							"isShadowHost":            false,
							"isDirectShadowHostChild": false,
							"pseudoClassLocks":        []any{},
							"mutationBreakpoints":     map[string]any{},
							"isDisplayed":             true,
							"isInHTMLDocument":        true,
							"traits":                  map[string]any{},
							"browsingContextID":       1,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getPageStyle":
				err := writePacket(writer, map[string]any{
					"from": message["to"],
					"pageStyle": map[string]any{
						"actor": "plex.conn0.pageStyle1",
						"traits": map[string]bool{
							"fontStretchLevel4": false,
							"fontStyleLevel4":   false,
							"fontVariations":    false,
							"fontWeightLevel4":  false,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getHighlighterByType":
				fallthrough
			default:
				err := writePacket(writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}

		case "plex.conn0.node1":
			fallthrough
		default:
			err := writePacket(writer, map[string]any{
				"from":    message["to"],
				"error":   "unrecognizedPacketType",
				"message": "unable to process request",
			})
			if err != nil {
				logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				return
			}
			logger.DebugContext(ctx, "sent packet")
		}

	}
}
