const baseColors = {
  native: "#326335",
  openconfig: "#38168C"
};

function shadeColor(col, percent) {
  const num = parseInt(col.slice(1), 16);
  let r = (num >> 16) + percent;
  let g = (num >> 8 & 0x00FF) + percent;
  let b = (num & 0x0000FF) + percent;
  r = Math.max(Math.min(255, r), 0);
  g = Math.max(Math.min(255, g), 0);
  b = Math.max(Math.min(255, b), 0);
  return "#" + (r << 16 | g << 8 | b).toString(16).padStart(6, "0");
}

// --------- Fonction principale de rendu ----------
function renderTree(data) {
  const svg = d3.select("#treeSvg");
  const width = window.innerWidth;
  const height = window.innerHeight;

  svg
    .attr("width", width)
    .attr("height", height);

  // Nettoyer l'ancien contenu
  svg.selectAll("*").remove();

  const g = svg.append("g")
    .attr("transform", `translate(${width / 2}, 40)`);

  const minNodeWidth = 220;
  const nodeHeight = 28;
  const lineHeight = 18;
  const levelGapY = 90;
  const horizontalPadding = 16;

  function buildHierarchy(data) {
    const root = {
      name: data.rootName,
      type: "root",
      color: "#000000",
      children: []
    };

    data.listOfPaths.forEach(path => {
      const baseColor = baseColors[path.origin] || "#888888";
      const listColor = shadeColor(baseColor, +30);

      const extraLines = [];
      if (typeof path.interval === "number") {
        extraLines.push(`Interval: ${path.interval} sec(s)`);
      }
      if (Array.isArray(path.aliases) && path.aliases.length > 0) {
        path.aliases.forEach(a => extraLines.push(`Alias: ${a}`));
      }

      const pathNode = {
        type: "path",
        name: path.name || "",
        origin: path.origin,
        color: baseColor,
        extraLines,
        children: []
      };

      if (path.listOfFields && path.listOfFields.length > 0) {
        const fieldsBox = {
          type: "fieldsBox",
          color: listColor,
          items: path.listOfFields.slice(),
          children: []
        };
        pathNode.children.push(fieldsBox);
      }

      root.children.push(pathNode);
    });

    return root;
  }

  const hierarchyData = buildHierarchy(data);
  const root = d3.hierarchy(hierarchyData);

  const treeLayout = d3.tree()
    .nodeSize([minNodeWidth + 20, levelGapY])
    .separation((a, b) => (a.parent === b.parent ? 1 : 0.9));

  treeLayout(root);

  // liens
  const link = g.selectAll(".link")
    .data(root.links())
    .join("path")
    .attr("class", "link")
    .attr("d", d => {
      const sx = d.source.x;
      const sHeight = getNodeHeight(d.source.data);
      const sy = d.source.y + sHeight / 2;

      const tx = d.target.x;
      const tHeight = getNodeHeight(d.target.data);
      const ty = d.target.y - tHeight / 2;

      const cx = (sx + tx) / 2;
      const cy1 = sy + (ty - sy) * 0.3;
      const cy2 = sy + (ty - sy) * 0.7;
      return `M${sx},${sy} C${sx},${cy1} ${tx},${cy2} ${tx},${ty}`;
    });

  // nœuds
  const node = g.selectAll(".node")
    .data(root.descendants())
    .join("g")
    .attr("class", "node")
    .attr("transform", d => `translate(${d.x},${d.y})`);

  node.append("rect")
    .attr("x", -minNodeWidth / 2)
    .attr("y", d => -getNodeHeight(d.data) / 2)
    .attr("width", minNodeWidth)
    .attr("height", d => getNodeHeight(d.data))
    .attr("rx", 8)
    .attr("ry", 8)
    .attr("fill", d => {
      const t = d.data.type;
      if (t === "root") return "#000000";
      if (d.data.color) return d.data.color;
      return "#444444";
    });

  node.each(function(d) {
    const group = d3.select(this);
    const data = d.data;

    if (data.type === "path") {
      const totalHeight = getNodeHeight(data);

      group.append("text")
        .attr("text-anchor", "middle")
        .attr("font-weight", "700")
        .attr("y", -totalHeight / 2 + lineHeight)
        .text(data.name || "");

      const leftX = -minNodeWidth / 2 + 8;
      if (Array.isArray(data.extraLines)) {
        data.extraLines.forEach((line, i) => {
          group.append("text")
            .attr("text-anchor", "start")
            .attr("x", leftX)
            .attr("y", -totalHeight / 2 + lineHeight * (2 + i))
            .text(line);
        });
      }
    } else if (data.type === "fieldsBox") {
      const totalHeight = getNodeHeight(data);
      const leftX = -minNodeWidth / 2 + 8;
      if (Array.isArray(data.items)) {
        data.items.forEach((item, i) => {
          group.append("text")
            .attr("text-anchor", "start")
            .attr("x", leftX)
            .attr("y", -totalHeight / 2 + lineHeight * (1 + i))
            .text(item);
        });
      }
    } else {
      group.append("text")
        .attr("text-anchor", "middle")
        .text(data.name || "");
    }
  });

  // ajuster largeur au texte le plus long
  node.each(function() {
    const group = d3.select(this);
    const rect = group.select("rect");
    const texts = group.selectAll("text").nodes();

    let maxWidth = 0;
    texts.forEach(t => {
      const w = t.getComputedTextLength();
      if (w > maxWidth) maxWidth = w;
    });

    const finalWidth = Math.max(minNodeWidth, maxWidth + horizontalPadding * 2);
    rect
      .attr("x", -finalWidth / 2)
      .attr("width", finalWidth);

    group.selectAll("text").each(function() {
      const txt = d3.select(this);
      const anchor = txt.attr("text-anchor");
      if (anchor === "middle") {
        txt.attr("x", 0);
      } else if (anchor === "start") {
        txt.attr("x", -finalWidth / 2 + 8);
      }
    });
  });

  // zoom / pan
  const zoom = d3.zoom()
    .scaleExtent([0.4, 2])
    .on("zoom", (event) => {
      g.attr("transform", event.transform);
    });

  svg.call(zoom)
    .call(zoom.transform, d3.zoomIdentity.translate(width / 2, 40));

  // helper local
  function getNodeHeight(data) {
    if (data.type === "root") {
      return nodeHeight;
    }
    if (data.type === "path") {
      const n = (data.extraLines && data.extraLines.length) || 0;
      const lines = 1 + n;
      return Math.max(nodeHeight, lines * lineHeight + lineHeight * 0.5);
    }
    if (data.type === "fieldsBox") {
      const n = (data.items && data.items.length) || 0;
      const lines = Math.max(1, n);
      return Math.max(nodeHeight, lines * lineHeight + lineHeight * 0.5);
    }
    return nodeHeight;
  }
}

function updateDoc() {
  var p = document.getElementById("profiles").value.trim();
  var img = document.getElementById("profileImg");
  var desc = document.getElementById("profileDesc");
  var tele = document.getElementById("profileTele");
  var graf = document.getElementById("profileGraf");
  var kapa = document.getElementById("profileKapa");

  if (p == "default") {
    img.setAttribute('src', "img/default.png");
    desc.innerHTML = "N/A";
    tele.innerHTML = "N/A";
    graf.innerHTML = "N/A";
    kapa.innerHTML = "N/A";
  } else {
    var dataToSend = {
      "profile": p
    };
    waitingDialog.show();
    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/updatedoc",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            //img.setAttribute('src', "img/" + json.img);
            desc.innerHTML = json.desc.trim();
            tele.innerHTML = json.tele.trim();
            graf.innerHTML = json.graf.trim();
            kapa.innerHTML = json.kapa.trim();
            waitingDialog.hide();

          } else {
            alertify.alert("JSTO...", json.msg);
            waitingDialog.hide();
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error");
          waitingDialog.hide();
        }
      });
    });
  }
}

async function loadConfig(fileName) {
  try {
    const response = await fetch(fileName);
    if (!response.ok) {
      throw new Error(`Failed to load ${fileName}: ${response.statusText}`);
    }

    const jsonContent = await response.json(); // Parse JSON

    // Pretty-print JSON
    const formattedJson = JSON.stringify(jsonContent, null, 2);

    // Add syntax highlighting
    const highlightedJson = Prism.highlight(formattedJson, Prism.languages.json, 'json');

    // Update modal content with highlighted JSON
    document.getElementById('modalcore').innerHTML = `<pre><code class="language-json">${highlightedJson}</code></pre>`;

    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('config'));
    modal.show();
  } catch (error) {
    alertify.alert("JSTO...", "Error loading config: " + error);
    document.getElementById('modalcore').textContent = 'Error loading configuration.';
  }
}

function showSensor(family, profile, config) {
  try {
    var dataToSend = {
      "family": family,
      "profile": profile,
      "config": config
    };

    waitingDialog.show();
    // send data
    $(function () {
      $.ajax({
        type: 'POST',
        url: "/gettree",
        data: JSON.stringify(dataToSend),
        contentType: "application/json",
        dataType: "json",
        success: function (json) {
          if (json.status == "OK") {
            renderTree(json.tree);
          } else {
            alertify.alert("JSTO...", json.msg);
            waitingDialog.hide();
          }
        },
        error: function (xhr, ajaxOptions, thrownError) {
          alertify.alert("JSTO...", "Unexpected error");
          waitingDialog.hide();
        }
      });
    });
  } catch (error) {
    alertify.alert("JSTO...", "Error loading tree: " + error);
    document.getElementById('modalcore').textContent = 'Error loading tree.';
  }
}


document.querySelector('#config .close').addEventListener('click', function () {
  const modal = document.getElementById('config');
  modal.classList.remove('show'); // Remove the `show` class
  modal.style.display = 'none';  // Hide the modal
  document.body.classList.remove('modal-open'); // Remove the modal-open class from body
  const backdrop = document.querySelector('.modal-backdrop');
  if (backdrop) backdrop.remove(); // Remove the backdrop element
});